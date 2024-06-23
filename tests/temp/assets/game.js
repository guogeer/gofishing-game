
function parseQuery() {
	const search = location.search.slice(1)
	const pairs = search ? search.split('&') : []
	const query = {}
	for (let i = 0; i < pairs.length; ++i) {
		const [key, value] = pairs[i].split('=')
		query[key] = query[key] || decodeURIComponent(value)
	}
	return query
}

function inArray(arr, target) {
	for (i = 0; i < arr.length; i++) {
		if (arr[i] == target) {
			return true
		}
	}
	return false
}

function encode(msg_id, msg_obj) {
	const default_sign = '12345678'
	const pkg = {
		"id": msg_id,
		"data": msg_obj,
		"sign": default_sign
	}

	const raw_body = JSON.stringify(pkg)
	const md5_sum = md5("hellokitty" + raw_body)

	let md5_part = ''
	const md5_indexs = [0, 3, 4, 8, 10, 11, 13, 14]
	md5_indexs.forEach(function (v) {
		md5_part = md5_part + md5_sum[v]
	})

	const last_index = raw_body.lastIndexOf(default_sign)
	const msg_body = raw_body.substring(0, last_index) + md5_part + raw_body.substring(last_index + md5_part.length)
	return msg_body
}