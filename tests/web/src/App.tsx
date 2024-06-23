import { Button, Flex, Form, Input } from 'antd';
import FingerGuessing from './fingerguessing';
import { useState } from 'react';
import { Account } from './utils/game';


const App = () => {
  const [form] = Form.useForm();
  const [accounts, setAccounts] = useState<Account[]>()
  return (
    <div className="App" style={{ margin: 20 }}>
      <Form
        layout="inline"
        form={form}
        initialValues={{ loginAddr: "http://localhost:9501", openId: "test1002" }}
      >
        <Form.Item
          label="登录地址"
          name="loginAddr"

        >
          <Input />
        </Form.Item>
        <Form.Item
          label="openId"
          name="openId"

        >
          <Input />
        </Form.Item>
        <Form.Item>
          <Button type="primary" onClick={(e) => {
            e.preventDefault()
            const addr = form.getFieldValue("loginAddr")
            const openId: string = form.getFieldValue("openId")

            const pattern = new RegExp('[0-9]+$')
            const index = openId.search(pattern)

            let nextOpenId = openId + "0"
            if (index >= 0) {
              nextOpenId = `${openId.slice(0, index)}${parseInt(openId.slice(index)) + 1}`
            }
            form.setFieldValue("openId", nextOpenId)
            console.log("xxxxxxxxxxxxxxx", index, openId.slice(index), nextOpenId, [{ loginAddr: addr, openId: openId }].concat(accounts || []))
            setAccounts([{ loginAddr: addr, openId: openId }].concat(accounts || []))
          }}>新建链接</Button>
        </Form.Item>
      </Form>
      <Flex gap="middle">
        {
          accounts?.map((account: Account) => <FingerGuessing key={account.openId} account={account} />)
        }
      </Flex>

    </div >
  )

}

export default App;