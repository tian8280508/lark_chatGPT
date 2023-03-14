package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"os/exec"
	"time"
)

const (
	address           = "0.0.0.0"
	port              = 0
	APPID             = ""
	APPSecret         = ""
	AppAccessTokenUrl = "https://open.feishu.cn/open-apis/auth/v3/app_access_token/internal"
)

type simpleHandler struct {
}

type userContent struct {
	Text string `json:"text"`
}

func (*simpleHandler) serveHTTP(w http.ResponseWriter, r *http.Request) {
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(requestDump))
	if r.Method == "GET" {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	decoder := json.NewDecoder(r.Body)
	event := &larkim.P2MessageReceiveV1{}
	if err := decoder.Decode(&event); err != nil {
		log.Println(err)
		return
	}

	var userContent *userContent
	if err := json.Unmarshal([]byte(*event.Event.Message.Content), &userContent); err != nil {
		log.Printf("serveHTTP Unmarshal error:%v", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	go func() {
		answer, err := callOpenAI(userContent.Text)
		if err != nil {
			log.Println(err)
			return
		}
		if err := sendMsg(*event.Event.Message.ChatId, *event.Event.Message.MessageId, answer); err != nil {
			log.Println(err)
		}
	}()
}

func sendMsg(receiveId, uuid, content string) error {
	url := "https://open.feishu.cn/open-apis/im/v1/messages"
	newContent := map[string]string{
		"text": content,
	}
	newContentMarshal, _ := json.Marshal(newContent)
	rand.Seed(time.Now().Unix())
	data := map[string]string{
		"receive_id": receiveId,
		"msg_type":   "text",
		"content":    string(newContentMarshal),
		"uuid":       uuid,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("sendMsg JSON marshal. Err:%s", err)
		return err
	}
	bytes := bytes.NewBuffer(jsonData)
	req, err := http.NewRequest("POST", url, bytes)
	if err != nil {
		log.Println(err)
		return err
	}
	token, _ := GetTenantAccessToken()
	q := req.URL.Query()
	q.Add("receive_id_type", "chat_id")
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Add("Content-Type", "application/json; charset=utf-8")
	//client := &http.Client{Timeout: 10 * time.Second}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("sendMsg DefaultClient error:%v", err)
		return err
	}
	log.Printf("sendMsg response Status:%v\n", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Printf("sendMsg response Body:%v\n", string(body))
	defer resp.Body.Close()

	return nil
}

func GetTenantAccessToken() (string, error) {
	data := map[string]string{
		"app_id":     APPID,
		"app_secret": APPSecret,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("GetTenantAccessToken JSON marshal. Err:%s", err)
		return "", err
	}
	bytes := bytes.NewBuffer(jsonData)
	req, err := http.NewRequest("POST", AppAccessTokenUrl, bytes)
	if err != nil {
		log.Println(err)
		return "", err
	}
	req.Header.Add("Content-Type", "application/json; charset=utf-8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("GetTenantAccessToken DefaultClient error:%v", err)
		return "", err
	}
	log.Printf("GetTenantAccessToken response Status:%v\n", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Printf("GetTenantAccessToken response Body:%v\n", string(body))
	respData := struct {
		AppAccessToken string `json:"app_access_token"`
		Code           int    `json:"code"`
	}{}
	if err := json.Unmarshal(body, &respData); err != nil {
		log.Printf("GetTenantAccessToken Unmarshal error:%v", err)
		return "", err
	}
	defer resp.Body.Close()
	return respData.AppAccessToken, nil
}

func (*simpleHandler) APITest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	var userContent *userContent
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return
	}
	if err := json.Unmarshal(body, &userContent); err != nil {
		log.Printf("APITest Unmarshal error:%v", err)
		return
	}
	res, err := callOpenAI(userContent.Text)
	if err != nil {
		log.Println(err)
		return
	}
	w.Write([]byte(res))
}

func main() {
	handler := &simpleHandler{}
	http.HandleFunc("/event", handler.serveHTTP)
	http.HandleFunc("/test", handler.APITest)
	log.Printf("Server started listening on %v", getHostAndPort(address, port))
	http.ListenAndServe(getHostAndPort(address, port), nil)
}

func callOpenAI(prompt string) (string, error) {
	if prompt == "" {
		return "", errors.New("empty prompt")
	}
	// 第三个参数整体是一个参数，不会因为中间有空格而变成多个参数
	cmd := exec.Command("python3", "openai_api.py", prompt)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("callOpenAI error:%v", err)
		return "", err
	}
	return string(out), err
}

func getHostAndPort(addr string, port int) string {
	return fmt.Sprintf("%v:%v", addr, port)
}

func HTTP404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func HTTP200(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	resp := make(map[string]string)
	resp["message"] = "Status OK"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}
	w.Write(jsonResp)
}
