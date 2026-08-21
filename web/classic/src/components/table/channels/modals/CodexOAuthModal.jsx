/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import { useTranslation } from 'react-i18next';
import { Modal, Button, Space, Banner } from '@douyinfe/semi-ui';

const CodexOAuthModal = ({ visible, onCancel }) => {
  const { t } = useTranslation();

  // Device authorization is channel-scoped, but this isolated classic modal
  // receives no saved channel id. Never fall back to the disabled PKCE flow.

  return (
    <Modal
      title={t('Codex 授权')}
      visible={visible}
      onCancel={onCancel}
      maskClosable
      closeOnEsc
      width={560}
      footer={
        <Space>
          <Button theme='solid' type='primary' onClick={onCancel}>
            {t('关闭')}
          </Button>
        </Space>
      }
    >
      <Banner
        type='warning'
        description={t(
          'Classic 模式不支持 Codex 授权。请使用默认界面为已保存的 Codex 渠道启动设备授权。此窗口不会修改渠道凭据。',
        )}
      />
    </Modal>
  );
};

export default CodexOAuthModal;
