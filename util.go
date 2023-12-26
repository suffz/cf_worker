package cf_worker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

func (CloudInfo *CloudRequest) Cloudflare() Cloud {
	if CloudInfo.WaitTime == 0 {
		CloudInfo.WaitTime = 15
	}
	if CloudInfo.Script == "" {
		CloudInfo.Script = BasicScript
	}
	run, err := playwright.Run()
	if err != nil {
		return Cloud{Err: err}
	}
	browser, err := run.Chromium.Launch()
	if err != nil {
		return Cloud{Err: err}
	}
	page, err := browser.NewPage()
	if err != nil {
		return Cloud{Err: err}
	}
	type Info struct {
		URL   string
		TOKEN string
		ERROR error
	}
	var token = make(chan Info, 1)
	page.Route("**/*", func(r playwright.Route) {
		ra := r.Request()
		if ra.URL() == "https://workers.cloudflare.com/playground/api/worker" {
			buffer, content_type := makeRequestBody(CloudInfo.Script, CloudInfo.JSFileName)
			ra := ra.Headers()
			ra["Content-Type"] = content_type
			r.Continue(playwright.RouteContinueOptions{Headers: ra, PostData: buffer.String()})
			return
		} else if strings.Contains(ra.URL(), "cloudflarepreviews.com/") && !strings.Contains(ra.URL(), ".update-preview-token") {
			for nam, val := range ra.Headers() {
				if nam == "x-cf-token" {
					token <- Info{
						URL:   ra.URL(),
						TOKEN: val,
					}
					break
				}
			}
		}
		r.Continue()
	})
	go func() {
		time.Sleep(time.Duration(CloudInfo.WaitTime) * time.Second)
		token <- Info{
			ERROR: errors.New("Waittime exceeded while fetching api token."),
		}
	}()
	page.Goto("https://workers.cloudflare.com/playground")
	user := <-token

	C := Cloud{
		ApiURL: user.URL,
		Token:  user.TOKEN,
		Body:   CloudInfo.Script,
		Cookie: struct {
			Name  string
			Value string
		}{
			Name:  "token",
			Value: user.TOKEN,
		},
		Err: user.ERROR,
	}

	if C.Err == nil {
		os.Mkdir("configs", 0644)
		path := fmt.Sprintf("configs/config_%v", time.Now().Unix())

		if CloudInfo.FileSave != "" {
			path = CloudInfo.FileSave
		}

		C.ConfigPath = path
		data, _ := json.MarshalIndent(C, "  ", "    ")
		os.WriteFile(path, data, 0644)
	}

	return C
}

func LoadConfig(path string) (Cloud, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Cloud{}, err
	}

	var C Cloud

	json.Unmarshal(data, &C)

	request, err := C.BuildRequestBase("GET")
	if err != nil {
		panic(err)
	}

	resp := C.Request(request)
	if resp.StatusCode == 200 {
		return C, nil
	}
	d, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(d), "error code") {
		return C, nil
	}

	InstallBrowsers()

	C = (&CloudRequest{
		FileSave:   C.ConfigPath,
		Script:     C.Body,
		JSFileName: "index.js",
		WaitTime:   15,
	}).Cloudflare()

	return C, C.Err
}

func SetupPaid(playground_api string) Cloud {
	if !strings.Contains(playground_api, "workers-playground") || !strings.Contains(playground_api, "workers.dev") {
		return Cloud{
			Err: errors.New("Error: Playground API URI Invalid."),
		}
	}

	C := Cloud{
		Paid:   true,
		ApiURL: playground_api,
	}

	if C.Err == nil {
		os.Mkdir("configs", 0644)
		path := fmt.Sprintf("configs/config_%v", time.Now().Unix())
		C.ConfigPath = path
		data, _ := json.MarshalIndent(C, "  ", "    ")
		os.WriteFile(path, data, 0644)
	}

	return C
}

func makeRequestBody(body, name string) (*bytes.Buffer, string) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	defer writer.Close()
	h := make(textproto.MIMEHeader)

	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, name, name))
	h.Set("Content-Type", "application/javascript+module")
	body_script, _ := writer.CreatePart(h)
	body_script.Write([]byte(body))

	h.Set("Content-Disposition", `form-data; name="metadata"; filename="blob"`)
	h.Set("Content-Type", "application/json")
	main_module, _ := writer.CreatePart(h)
	main_module.Write([]byte(fmt.Sprintf(`{"main_module":"%v"}`, name)))

	return buf, writer.FormDataContentType()
}

func (C *Cloud) BuildRequestBase(METHOD string) (*http.Request, error) {
	if C.ApiURL == "" {
		return nil, errors.New("No apiURL found")
	}
	req, err := http.NewRequest(METHOD, C.ApiURL, nil)
	if err != nil {
		return nil, err
	}
	if !C.Paid {
		req.AddCookie(&http.Cookie{Name: C.Cookie.Name, Value: C.Cookie.Value})
	}
	return req, nil
}

func (C *Cloud) ApplyDataBody(r *http.Request, body []byte) *http.Request {
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	return r
}

func (C *Cloud) Request(req *http.Request) *http.Response {
	return send(req, C)
}

func send(req *http.Request, C *Cloud) *http.Response {
	resp, _ := http.DefaultClient.Do(req)
	return resp
}

func InstallBrowsers() {
	playwright.Install()
}
