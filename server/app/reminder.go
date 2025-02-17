// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"
)

const RetrospectivePrefix = "retro_"

// HandleReminder is the handler for all reminder events.
func (s *PlaybookRunServiceImpl) HandleReminder(key string) {
	if strings.HasPrefix(key, RetrospectivePrefix) {
		s.handleReminderToFillRetro(strings.TrimPrefix(key, RetrospectivePrefix))
	} else {
		s.handleStatusUpdateReminder(key)
	}
}

func (s *PlaybookRunServiceImpl) handleReminderToFillRetro(playbookRunID string) {
	playbookRunToRemind, err := s.GetPlaybookRun(playbookRunID)
	if err != nil {
		s.logger.Errorf(errors.Wrapf(err, "handleReminderToFillRetro failed to get playbook run id: %s", playbookRunID).Error())
		return
	}

	// In the meantime we did publish a retrospective, so no reminder.
	if playbookRunToRemind.RetrospectivePublishedAt != 0 {
		return
	}

	// If we are not in the resolved state then don't remind
	if playbookRunToRemind.CurrentStatus != StatusResolved &&
		playbookRunToRemind.CurrentStatus != StatusArchived {
		return
	}

	if err = s.postRetrospectiveReminder(playbookRunToRemind, false); err != nil {
		s.logger.Errorf(errors.Wrapf(err, "couldn't post reminder").Error())
		return
	}

	// Jobs can't be rescheduled within themselves with the same key. As a temporary workaround do it in a delayed goroutine
	go func() {
		time.Sleep(time.Second * 2)
		if err = s.SetReminder(RetrospectivePrefix+playbookRunID, time.Duration(playbookRunToRemind.RetrospectiveReminderIntervalSeconds)*time.Second); err != nil {
			s.logger.Errorf(errors.Wrap(err, "failed to reocurr retrospective reminder").Error())
			return
		}
	}()
}

func (s *PlaybookRunServiceImpl) handleStatusUpdateReminder(playbookRunID string) {
	playbookRunToModify, err := s.GetPlaybookRun(playbookRunID)
	if err != nil {
		s.logger.Errorf(errors.Wrapf(err, "HandleReminder failed to get playbook run id: %s", playbookRunID).Error())
		return
	}

	owner, err := s.pluginAPI.User.Get(playbookRunToModify.OwnerUserID)
	if err != nil {
		s.logger.Errorf(errors.Wrapf(err, "HandleReminder failed to get owner for id: %s", playbookRunToModify.OwnerUserID).Error())
		return
	}

	attachments := []*model.SlackAttachment{
		{
			Actions: []*model.PostAction{
				{
					Type: "button",
					Name: "Update status",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/v0/runs/%s/reminder/button-update",
							s.configService.GetManifest().Id,
							playbookRunToModify.ID),
					},
				},
				{
					Type: "button",
					Name: "Dismiss",
					Integration: &model.PostActionIntegration{
						URL: fmt.Sprintf("/plugins/%s/api/v0/runs/%s/reminder/button-dismiss",
							s.configService.GetManifest().Id,
							playbookRunToModify.ID),
					},
				},
			},
		},
	}

	post, err := s.poster.PostMessageWithAttachments(playbookRunToModify.ChannelID, attachments,
		"@%s, please provide a status update.", owner.Username)
	if err != nil {
		s.logger.Errorf(errors.Wrap(err, "HandleReminder error posting reminder message").Error())
		return
	}

	playbookRunToModify.ReminderPostID = post.Id
	if err = s.store.UpdatePlaybookRun(playbookRunToModify); err != nil {
		s.logger.Errorf(errors.Wrapf(err, "error updating with reminder post id, playbook run id: %s", playbookRunToModify.ID).Error())
	}
}

// SetReminder sets a reminder. After timeInMinutes in the future, the owner will be
// reminded to update the playbook run's status.
func (s *PlaybookRunServiceImpl) SetReminder(playbookRunID string, fromNow time.Duration) error {
	if _, err := s.scheduler.ScheduleOnce(playbookRunID, time.Now().Add(fromNow)); err != nil {
		return errors.Wrap(err, "unable to schedule reminder")
	}

	return nil
}

// RemoveReminder removes the pending reminder for the given playbook run, if any.
func (s *PlaybookRunServiceImpl) RemoveReminder(playbookRunID string) {
	s.scheduler.Cancel(playbookRunID)
}

// RemoveReminderPost removes the reminder post in the channel for the given playbook run, if any.
func (s *PlaybookRunServiceImpl) RemoveReminderPost(playbookRunID string) error {
	playbookRunToModify, err := s.store.GetPlaybookRun(playbookRunID)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve playbook run")
	}

	return s.removeReminderPost(playbookRunToModify)
}

// removeReminderPost removes the reminder post in the channel for the given playbook run, if any.
func (s *PlaybookRunServiceImpl) removeReminderPost(playbookRunToModify *PlaybookRun) error {
	if playbookRunToModify.ReminderPostID == "" {
		return nil
	}

	post, err := s.pluginAPI.Post.GetPost(playbookRunToModify.ReminderPostID)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve reminder post")
	}

	if post.DeleteAt != 0 {
		return nil
	}

	if err = s.pluginAPI.Post.DeletePost(playbookRunToModify.ReminderPostID); err != nil {
		return errors.Wrapf(err, "failed to delete reminder post")
	}

	playbookRunToModify.ReminderPostID = ""
	if err = s.store.UpdatePlaybookRun(playbookRunToModify); err != nil {
		return errors.Wrapf(err, "failed to update playbook run after removing reminder post id")
	}

	return nil
}

// ResetReminderTimer sets the previous reminder timer to 0.
func (s *PlaybookRunServiceImpl) ResetReminderTimer(playbookRunID string) error {
	playbookRunToModify, err := s.store.GetPlaybookRun(playbookRunID)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve playbook run")
	}

	playbookRunToModify.PreviousReminder = 0
	if err = s.store.UpdatePlaybookRun(playbookRunToModify); err != nil {
		return errors.Wrapf(err, "failed to update playbook run after resetting reminder timer")
	}

	s.poster.PublishWebsocketEventToChannel(playbookRunUpdatedWSEvent, playbookRunToModify, playbookRunToModify.ChannelID)

	return nil
}
