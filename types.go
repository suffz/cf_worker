package cf_worker

var BasicScript = `export default {
	/**
	 * @param {Request} request
	 */
	async fetch(request) {
		let opts = await request.json();
		if (opts["url"]) {
			let req_options = {}
			if (opts["method"] == "GET" || opts["method"] == "HEAD") {
				let headers = getHeaderObj(opts["headers"])
				req_options = {
					redirect: 'follow',
					method: opts["method"],
					headers: headers,
				}
			} else {
				let headers = getHeaderObj(opts["headers"])
				req_options = {
					redirect: 'follow',
					method: opts["method"],
					body: 	opts["body"],
					headers: headers,
				}
			}
			let r = await fetch(opts["url"], req_options);

			let Headers = []

			r.headers.forEach((value, name) => {
				Headers.push({name: name, value: value})
			})

			let options = {true: async function() {
				let ra = await r.arrayBuffer()
				return _arrayBufferToBase64(ra)
			}, false: async function() {
				let ra = await r.text()
				return ra
			}}

			return Response.json({
				"headers":Headers,
				"url_sent":r.url,
				"status":r.status,
				"body":await options[(String(opts["url"]).includes(".png"))](),
			})
		}
	},
};

function getHeaderObj(o) {
	var obj = {};
	for (var value of o) {
		obj[value["name"]] = value["value"]
	}
	return obj;
}

function _arrayBufferToBase64( buffer ) {
    var binary = '';
    var bytes = new Uint8Array( buffer );
    var len = bytes.byteLength;
    for (var i = 0; i < len; i++) {
        binary += String.fromCharCode( bytes[ i ] );
    }
    return btoa( binary );
}`

type Cloud struct {
	Paid       bool
	ConfigPath string
	ApiURL     string
	Token      string
	Body       string
	Cookie     struct {
		Name  string
		Value string
	}
	Err error
}

type CloudRequest struct {
	FileSave   string // path to save the config to
	Script     string // will use detault BasicScript variable
	JSFileName string
	WaitTime   int // Defaults To 15
}

//

type Reply struct {
	URLSent string         `json:"url_sent"`
	Status  int            `json:"status"`
	Body    string         `json:"body"`
	Headers []HeadersReply `json:"headers"`
}

type HeadersReply struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type WorkerRequest struct {
	URL     string    `json:"url"`
	Method  string    `json:"method"`
	Body    string    `json:"body"`
	Headers []Headers `json:"headers"`
}
type Headers struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
