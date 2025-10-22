import React, { useState } from 'react';
import { Modal, Checkbox, Typography, Divider } from 'antd';
import { ExclamationCircleOutlined } from '@ant-design/icons';

const { Title, Paragraph, Text } = Typography;

interface CybersecurityLawModalProps {
  visible: boolean;
  onAccept: () => void;
  onReject: () => void;
}

const CybersecurityLawModal: React.FC<CybersecurityLawModalProps> = ({
  visible,
  onAccept,
  onReject,
}) => {
  const [agreed, setAgreed] = useState(false);

  const handleAccept = () => {
    if (agreed) {
      onAccept();
    }
  };

  const handleReject = () => {
    onReject();
  };

  return (
    <Modal
      title={
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <ExclamationCircleOutlined style={{ color: '#faad14' }} />
          <span>网络安全法协议确认</span>
        </div>
      }
      open={visible}
      onOk={handleAccept}
      onCancel={handleReject}
      okText="我同意并遵守"
      cancelText="不同意，退出程序"
      okButtonProps={{ disabled: !agreed }}
      cancelButtonProps={{ danger: true }}
      width={600}
      closable={false}
      maskClosable={false}
    >
      <div style={{ maxHeight: '60vh', overflowY: 'auto' }}>
        <Title level={4}>重要提示</Title>
        <Paragraph>
          在使用本漏洞扫描工具之前，请您仔细阅读并遵守以下法律法规：
        </Paragraph>

        <Divider />

        <Title level={5}>《中华人民共和国网络安全法》相关条款</Title>
        <Paragraph>
          <Text strong>第二十七条：</Text>任何个人和组织不得从事非法侵入他人网络、干扰他人网络正常功能、窃取网络数据等危害网络安全的活动；不得提供专门用于从事危害网络安全活动的程序、工具；明知他人从事危害网络安全的活动的，不得为其提供技术支持、广告推广、支付结算等帮助。
        </Paragraph>

        <Paragraph>
          <Text strong>第四十六条：</Text>任何个人和组织应当对其使用网络的行为负责，不得设立用于实施诈骗，传授犯罪方法，制作或者销售违禁物品、管制物品等违法犯罪活动的网站、通讯群组，不得利用网络发布涉及实施诈骗，制作或者销售违禁物品、管制物品以及其他违法犯罪活动的信息。
        </Paragraph>

        <Divider />

        <Title level={5}>使用须知</Title>
        <Paragraph>
          <Text strong>1. 合法使用：</Text>本工具仅用于合法的安全测试、漏洞评估和网络安全研究目的。
        </Paragraph>
        <Paragraph>
          <Text strong>2. 授权测试：</Text>您只能在获得明确授权的系统上进行安全测试，不得对未授权的系统进行扫描。
        </Paragraph>
        <Paragraph>
          <Text strong>3. 数据保护：</Text>在测试过程中发现的数据和信息，应当妥善保管，不得泄露或用于非法目的。
        </Paragraph>
        <Paragraph>
          <Text strong>4. 责任承担：</Text>您应当对使用本工具的行为承担全部法律责任。
        </Paragraph>

        <Divider />

        <Title level={5}>免责声明</Title>
        <Paragraph>
          本工具仅作为安全研究和测试的辅助工具，开发者不对使用本工具可能产生的任何法律后果承担责任。用户应当确保其使用行为符合相关法律法规的要求。
        </Paragraph>

        <Divider />

        <div style={{ 
          padding: '16px', 
          backgroundColor: '#f6ffed', 
          border: '1px solid #b7eb8f',
          borderRadius: '6px',
          marginTop: '16px'
        }}>
          <Checkbox 
            checked={agreed} 
            onChange={(e) => setAgreed(e.target.checked)}
            style={{ fontSize: '14px' }}
          >
            <Text strong>
              我已仔细阅读并理解上述条款，承诺遵守《中华人民共和国网络安全法》等相关法律法规，
              仅将本工具用于合法的安全测试和研究目的，并承担相应的法律责任。
            </Text>
          </Checkbox>
        </div>
      </div>
    </Modal>
  );
};

export default CybersecurityLawModal;
