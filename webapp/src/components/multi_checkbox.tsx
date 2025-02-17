// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import styled, {StyledComponentBase} from 'styled-components';

import DotMenu, {DotMenuButton} from 'src/components/dot_menu';
import {CheckboxContainer} from 'src/components/checklist_item';

const IconWrapper = styled.div`
    display: inline-flex;
    padding: 0 4px;
`;

const FilterCheckboxContainer = styled(CheckboxContainer)`
    margin: 0 34px 0 20px;
    line-height: 32px;
    align-items: center;

    input[type='checkbox'] {
        width: 16px;
        min-width: 16px;
        height: 16px;
        border: 1px solid var(--center-channel-color-24);
        border-radius: 2px;
    }

    input[type="checkbox"]:checked:disabled {
        background: var(--button-bg-24);
        border: 1px solid var(--button-bg-24);
    }
`;

const OptionDisplay = styled.div`
    margin: 0;
`;

const Divider = styled.div`
    background: var(--center-channel-color-08);
    height: 1px;
    margin: 8px 0;
`;

const Title = styled.div`
    margin: 0 0 0 20px;
    font-weight: 600;
    font-size: 12px;
    line-height: 28px;
    color: var(--center-channel-color-56);
`;

export interface CheckboxOption {
    display: string;
    value: string;
    selected?: boolean;
    disabled?: boolean;
}

interface Props {
    options: CheckboxOption[];
    onselect: (value: string, checked: boolean) => void;
    dotMenuButton?: StyledComponentBase<'div', any>;
    icon?: JSX.Element;
}

const MultiCheckbox = (props: Props) => (
    <DotMenu
        dotMenuButton={props.dotMenuButton ?? DotMenuButtonRight}
        icon={
            props.icon ??
            <IconWrapper>
                <i className='icon icon-filter-variant'/>
            </IconWrapper>
        }
        wide={true}
    >
        {props.options.map((option, idx) => {
            if (option.value === 'divider') {
                return <Divider key={'divider' + idx}/>;
            }
            if (option.value === 'title') {
                return <Title key={'title' + idx}>{option.display}</Title>;
            }

            const onClick = () => props.onselect(option.value, !option.selected);

            return (
                <FilterCheckboxContainer
                    key={option.value}
                    onClick={onClick}
                >
                    <input
                        type='checkbox'
                        checked={option.selected}
                        disabled={option.disabled}
                        onChange={onClick}
                    />
                    <OptionDisplay>{option.display}</OptionDisplay>
                </FilterCheckboxContainer>
            );
        })}
    </DotMenu>
);

const DotMenuButtonRight = styled(DotMenuButton)`
    margin: 0 16px 0 auto;
`;

export default MultiCheckbox;
