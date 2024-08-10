import React, { useEffect, useState } from 'react';
import { Button, Descriptions, DescriptionsProps, Radio, Space } from 'antd';
import { Account, encode, inArray } from './utils/game';

type UserInfo = {
	uid: number
	nickname: string
	ip: string
	chip: number
	gesture: string
}

type SeatPlayer = {
	seatIndex: number
	uid: number
	nickname: string
	chip: number
	cmp: number
	gesture: string
	ip: string
}

type BaseRoomInfo = {
	status: number
	countdown: number
}

type RoomInfo = {
	seatPlayers: (SeatPlayer | null)[]
	gesture: string
} & BaseRoomInfo

type FingerGuessingUser = {
	roomInfo?: RoomInfo
} & UserInfo

type FingerGuessingProps = {
	account?: Account
}

type UserContext = {
	token: string
	serverId: string
	gwAddr: string
	gatewayConn?: WebSocket
}

type messageHandler = (args: any) => void;

type countdownTimer = {
	ts: number
	secs: number
}

const FingerGuessing: React.FC<FingerGuessingProps> = (
	{ account }
) => {
	const [countdownTimer, setCountdownTimer] = useState<countdownTimer>({ ts: 0, secs: 0 })
	const [userContext, setUserContext] = useState<UserContext>({ gwAddr: '', token: '', serverId: '' })
	const [currentUser, setCurrentUser] = useState<FingerGuessingUser>({
		uid: 0,
		nickname: '',
		ip: '',
		chip: 0,
		gesture: '',
		roomInfo: {
			status: 0,
			countdown: 0,
			seatPlayers: [],
			gesture: ''
		}
	})

	const ignoreMsgs = ['heartbeat']
	const login = () => {
		if (!account) {
			return
		}
		console.log("login addr", account.loginAddr, "openId", account.openId)
		fetch(account.loginAddr + "/api/v1/login", {
			method: 'POST',
			headers: {
				'Accept': 'application/json',
				'Content-Type': 'application/json'
			},
			body: encode('login', { 'openId': account.openId, 'plate': 'test', 'nickname': 'n_' + account.openId }),
		}).then(res => res.json())
			.then(res => {
				const wsAddr = `ws://${res.data.addr}/ws`
				console.log("login successfully websoket addr", wsAddr, "token", res.data.token)
				const conn = new WebSocket(wsAddr);

				setUserContext({ ...userContext, token: res.data.token, gwAddr: wsAddr, gatewayConn: conn })
			})
	}
	useEffect(() => {
		login()
	}, [account])

	const openCb = function () {
		console.log("connect gateway successfully", userContext.gwAddr, "token", userContext.token)
	}
	const messageCb = function (event: MessageEvent) {
		try {
			const obj = JSON.parse(event.data)
			if (!inArray(ignoreMsgs, obj.id.toLowerCase())) {
				console.log("recv msg", obj)
			}
			const h = userEvents.handlers[obj.id]
			if (h) h(obj.data)
		} catch (error) {
			console.error('recv invalid data', error, event.data)
		}
	}
	const errorCb = function () {
		setUserContext({ ...userContext, gatewayConn: undefined, serverId: '' })
		console.log("connect error");
	}

	useEffect(() => {
		const conn = userContext.gatewayConn

		if (conn) {
			// Connection opened
			conn.addEventListener("open", openCb);

			// Listen for messages
			conn.addEventListener("message", messageCb)

			conn.addEventListener("error", errorCb)
			return () => {
				conn.removeEventListener("message", messageCb)
				conn.removeEventListener("open", openCb)
				conn.removeEventListener("error", errorCb)
			}
		}
	}, [userContext, currentUser])

	useEffect(() => {
		const h = setInterval(userEvents.updateRoomCountdown, 1000)
		return () => {
			return clearInterval(h)
		}
	}, [countdownTimer.ts])

	const userEvents = {
		result: {
			"1": '赢', "0": '平', "-1": '输', "99": "空",
		},
		requests: {
			chooseGesture: (gesture: string) => {
				userEvents.sendMsg("fingerGuessing.chooseGesture", { "gesture": gesture })
			},
			enterGame: (name: string, subId: number = 0) => {
				userEvents.sendMsg(`${name}.enter`, { "token": userContext.token, "subId": subId, "leaveServer": userContext.serverId })
			},
			leave: () => {
				userEvents.sendMsg(`${userContext.serverId}.leave`, {})
			},
		},
		updateRoomCountdown: () => {
			if (userContext.serverId !== "fingerGuessing") {
				return
			}

			const current_date = new Date()
			const secs = countdownTimer.ts - current_date.getTime() / 1000
			setCountdownTimer({ ts: countdownTimer.ts, secs: Math.ceil(Math.max(secs, 0)) })
		},
		timer: 0,
		handlers: {
			"hall.enter": (args: any) => {
				if (args.code != "ok") {
					window.alert("enter game error: " + args.msg)
					return
				}
				setUserContext({ ...userContext, serverId: "hall" })
			},
			"hall.leave": (_: any) => {
				setUserContext({ ...userContext, serverId: "" })
			},
			"fingerGuessing.leave": (_: any) => {
				setUserContext({ ...userContext, serverId: "" })
			},
			"fingerGuessing.enter": (args: any) => {
				if (args.code != "ok") {
					window.alert("enter game error: " + args.msg)
					return
				}
				setUserContext({ ...userContext, serverId: "fingerGuessing" })
			},
			"hall.getUserInfo": (args: any) => {
				setUserContext({ ...userContext, serverId: "hall" })

				args.chip = 0
				args.items.forEach((item: any) => {
					if (item.id == 1001) {
						args.chip = item.num
					}
				})
				setCurrentUser({ ...args, ...args.baseInfo })
			},
			"fingerGuessing.getUserInfo": (args: any) => {
				args.chip = 0
				args.items.forEach((item: any) => {
					if (item.id == 1001) {
						args.chip = item.num
					}
				})
				let seatPlayers = []
				for (let i = 0; i < 4; i++) {
					seatPlayers.push(null)
				}
				args.roomInfo.seatPlayers.forEach((user: SeatPlayer) => {
					user.cmp = 99
					seatPlayers[user.seatIndex] = user
				})
				args.roomInfo.seatPlayers = seatPlayers
				Object.assign(currentUser, { ...args, ...args.baseInfo })

				userEvents.updateRoomCountdown()
			},
			"fingerGuessing.startGame": (args: any) => {
				if (!currentUser?.roomInfo) {
					return
				}
				currentUser.roomInfo.gesture = ''
				currentUser.roomInfo.status = 1
				currentUser.roomInfo.countdown = args.countdown
				currentUser.roomInfo.seatPlayers?.forEach((user: SeatPlayer | null) => {
					if (!user) return
					user.cmp = 99
					user.gesture = ""
				})
				countdownTimer.ts = args.countdown
				setCurrentUser({ ...currentUser })
				userEvents.updateRoomCountdown()
			},
			"fingerGuessing.gameOver": (args: any) => {
				if (undefined === currentUser.roomInfo) {
					return
				}

				currentUser.roomInfo.status = 0
				currentUser.roomInfo.countdown = args.countdown
				countdownTimer.ts = args.countdown

				args.result.forEach((item: SeatPlayer) => {
					let seatUser = currentUser.roomInfo?.seatPlayers[item.seatIndex]
					if (seatUser) {
						seatUser.cmp = item.cmp
					}
				})

				currentUser.gesture = ""
				currentUser.roomInfo.gesture = args.gesture
				setCurrentUser({ ...currentUser })
				userEvents.updateRoomCountdown()
			},
			"fingerGuessing.sitDown": (args: any) => {
				if (undefined === currentUser.roomInfo) {
					return
				}
				currentUser.roomInfo.seatPlayers[args.seatIndex] = {
					uid: args.uid,
					seatIndex: args.seatIndex,
					nickname: args.userInfo.nickname,
					chip: args.userInfo.chip,
					gesture: "",
					cmp: 99,
					ip: args.ip
				}
				setCurrentUser({ ...currentUser })
			},
			"fingerGuessing.sitUp": (args: any) => {
				if (undefined === currentUser.roomInfo) {
					return
				}
				currentUser.roomInfo.seatPlayers[args.seatIndex] = null
				setCurrentUser({ ...currentUser })
			},
			"fingerGuessing.chooseGesture": (args: any) => {
				if (undefined === currentUser.roomInfo) {
					return
				}
				currentUser.roomInfo.seatPlayers.forEach((seatUser: any) => {
					if (seatUser && seatUser.uid == args.uid) {
						seatUser.gesture = args.gesture
					}
				})
				if (args.uid == currentUser.uid) {
					currentUser.gesture = args.gesture
				}
				setCurrentUser({ ...currentUser })
			},
			"fingerGuessing.addItems": (args: any) => {
				let match_user: SeatPlayer | null = null
				if (undefined === currentUser.roomInfo) {
					return
				}
				currentUser.roomInfo?.seatPlayers.forEach((user: SeatPlayer | null) => {
					if (user && user.uid == args.uid) {
						match_user = user
					}
				})
				args.items.forEach((item: any) => {
					if (item.id == 1001) {
						if (match_user) {
							match_user.chip = match_user.chip + item.num

						}
						if (args.uid == currentUser.uid) {
							currentUser.chip += item.num
						}
					}
				})

				setCurrentUser({ ...currentUser })
			},

		} as { [msg_id: string]: messageHandler },

		sendMsg: (msg_id: string, msg_data: any) => {
			if (!userContext.gatewayConn) return

			try {
				const msg_obj = msg_data
				const msg_body = encode(msg_id, msg_obj)
				console.log("send msg", msg_id, msg_obj)
				userContext.gatewayConn.send(msg_body)
			} catch (error) {
				console.error("invalid json data", error)
			}
		},
		onConnect: () => {
			setInterval(() => {
				if (!userContext.gatewayConn) return
				userEvents.sendMsg('heartbeat', {})
			}, 5000)
		}
	}

	const userInfoItems: DescriptionsProps['items'] = [
		{
			key: 'personInfoUid',
			label: "uid",
			children: currentUser.uid,
		},
		{
			key: 'personInfoNickname',
			label: "昵称",
			children: currentUser.nickname,
		},
		{
			key: 'personInfoChip',
			label: "金币数",
			children: currentUser.chip,
		},
		{
			key: 'personInfoIp',
			label: "IP地址",
			children: currentUser.ip,
		},
		{
			key: 'personInfoGesture',
			label: "手势",
			children: currentUser.gesture,
		},
		{
			key: 'personInfoOpenId',
			label: "openId",
			children: account?.openId,
		},
	]
	const roomInfoItems: DescriptionsProps['items'] = [{
		key: 'roomCountdown',
		label: "倒计时",
		children: countdownTimer.secs,
	},
	{
		key: 'personInfoNickname',
		label: "状态",
		children: currentUser.roomInfo?.status == 0 ? "空闲中" : "游戏中",
	}]
	const getSeatItems = (seatPlayer: SeatPlayer | null) => {
		return [
			{
				key: 'seatUid',
				label: "uid",
				children: seatPlayer?.uid,
			},
			{
				key: 'seatNickname',
				label: "昵称",
				children: seatPlayer?.nickname,
			},
			{
				key: 'seatChip',
				label: "金币数",
				children: seatPlayer?.chip,
			},
			{
				key: 'seatIp',
				label: "IP地址",
				children: seatPlayer?.ip,
			},
			{
				key: 'seatGesture',
				label: "手势",
				children: seatPlayer?.gesture,
			}
		] as DescriptionsProps['items']
	}

	return (
		<div style={{ width: "33%", display: "inline" }}>
			<h4>入口</h4>
			<Space>
				<Button type="primary" onClick={() => { userEvents.requests.enterGame('fingerGuessing', 1001) }} disabled={userContext.serverId === 'fingerGuessing'}>房间</Button>
				<Button type="primary" onClick={() => { userEvents.requests.enterGame('hall') }} disabled={userContext.serverId === 'hall'}>大厅</Button>
				<Button type="primary" onClick={() => { userEvents.requests.leave() }} disabled={userContext.serverId !== ''}>离开</Button>
			</Space >
			{
				userContext.serverId === "fingerGuessing" && <>
					<h4>操作</h4>
					<Radio.Group buttonStyle="solid" disabled={currentUser.roomInfo?.status == 0 || currentUser.gesture !== ''}>
						<Radio.Button value="rock" onClick={() => { userEvents.requests.chooseGesture("rock") }}>石头</Radio.Button>
						<Radio.Button value="scissor" onClick={() => { userEvents.requests.chooseGesture("scissor") }}>剪刀</Radio.Button>
						<Radio.Button value="paple" onClick={() => { userEvents.requests.chooseGesture("paple") }}>布</Radio.Button>
					</Radio.Group >
				</>
			}
			<Descriptions title="个人信息" items={userInfoItems} />
			{
				currentUser.roomInfo && <Descriptions title="房间信息" items={roomInfoItems} />
			}
			{
				currentUser.roomInfo?.seatPlayers.map((seatPlayer: SeatPlayer | null, seatIndex: number) => {
					return <Descriptions key={`seat_${seatIndex}`} title={`座位${seatIndex}`} items={getSeatItems(seatPlayer)} />
				})
			}
		</div >
	)
}

export default FingerGuessing;
