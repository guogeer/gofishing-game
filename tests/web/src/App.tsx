import { Button, Flex, Form, Input } from 'antd';
import FingerGuessing from './fingerguessing';
import { useState } from 'react';
import { Account } from './utils/game';


const App = () => {
  const [form] = Form.useForm();
  const [accounts, setAccounts] = useState<Account[]>([])

  const accTable: Account[][] = []
  for (let i = 0; accounts && i * 4 < accounts.length; i++) {
    accTable.push(accounts.slice(4 * i, Math.min(4 * i + 4, accounts.length)))
  }

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
            const addr: string = form.getFieldValue("loginAddr")
            const openId: string = form.getFieldValue("openId")

            const pattern = new RegExp('[0-9]+$')
            const index = openId.search(pattern)

            let nextOpenId = openId + "0"
            if (index >= 0) {
              nextOpenId = `${openId.slice(0, index)}${parseInt(openId.slice(index)) + 1}`
            }
            form.setFieldValue("openId", nextOpenId)
            setAccounts(accounts.concat({ loginAddr: addr, openId: openId }))
          }}>新建链接</Button>
        </Form.Item>
      </Form>
      {
        accounts.length > 0 && accTable?.map((row, rowindex) =>
          <Flex key={`accountRow${rowindex}`} gap="middle">
            {
              row?.map((account: Account) => <FingerGuessing key={`user_${account.openId}`} account={account} />)
            }
          </Flex>

        )
      }


    </div >
  )

}

export default App;