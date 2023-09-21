package hw4_test_coverage

import (
	"encoding/json"
	//"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	//"reflect"
	"encoding/xml"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// код писать тут

var badJSON = `{"test":0`

type XmlUser struct {
	Id        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type XmlData struct {
	XMLName xml.Name  `xml:"root"`
	Users   []XmlUser `xml:"row"`
}

func parseXML() []User {
	xmlFile, _ := os.Open("dataset.xml")
	defer xmlFile.Close()
	data, _ := ioutil.ReadAll(xmlFile)
	var root XmlData
	xml.Unmarshal(data, &root)
	users := make([]User, len(root.Users))
	for i, u := range root.Users {
		users[i] = User{
			Id:     u.Id,
			Name:   u.FirstName + u.LastName,
			Age:    u.Age,
			About:  u.About,
			Gender: u.Gender,
		}
	}
	return users
}

func SearchServerErrors(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	offsetStr := query.Get("offset")
	offset, _ := strconv.Atoi(offsetStr)
	if offset == 1 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if offset == 2 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(badJSON))
		return
	} else if offset == 3 {
		w.WriteHeader(http.StatusBadRequest)
		resp := SearchErrorResponse{
			Error: "ErrorBadOrderField",
		}
		respJson, _ := json.Marshal(resp)
		w.Write(respJson)
		return
	} else if offset == 4 {
		w.WriteHeader(http.StatusBadRequest)
		resp := SearchErrorResponse{
			Error: "Other error",
		}
		respJson, _ := json.Marshal(resp)
		w.Write(respJson)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func SearchServerTimeout(w http.ResponseWriter, r *http.Request) {
	time.Sleep(1100 * time.Millisecond)
	w.WriteHeader(http.StatusGatewayTimeout)
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("AccessToken")
	if token != "auth112" {
		w.WriteHeader(http.StatusUnauthorized)
	}
	users := parseXML()

	var req SearchRequest
	query := r.URL.Query()
	if len(query) > 0 {
		st := query.Get("query")
		if len(st) > 0 {
			req.Query = st
		}
		st = query.Get("offset")
		if len(st) > 0 {
			req.Offset, _ = strconv.Atoi(st)
		}
		st = query.Get("limit")
		if len(st) > 0 {
			req.Limit, _ = strconv.Atoi(st)
		}
		st = query.Get("order_by")
		if len(st) > 0 {
			req.OrderBy, _ = strconv.Atoi(st)
		}
		st = query.Get("order_field")
		if len(st) > 0 {
			if st == "Id" || st == "Age" || st == "Name" {
				req.OrderField = st
			} else {
				w.WriteHeader(http.StatusBadRequest)
				resp := SearchErrorResponse{
					Error: "ErrorBadOrderField",
				}
				respJson, _ := json.Marshal(resp)
				w.Write(respJson)
			}
		} else {
			req.OrderField = "Name"
		}
	}
	result := make([]User, 0)
	for _, u := range users {
		if req.Query != "" {
			if !(strings.Contains(u.Name, req.Query) || strings.Contains(u.About, req.Query)) {
				continue
			}
		}
		result = append(result, u)
	}
	if req.OrderBy != 0 {
		sort.Slice(result, func(i, j int) bool {
			switch req.OrderField {
			case "Age":
				if req.OrderBy > 0 {
					return result[i].Age < result[j].Age
				} else {
					return result[i].Age > result[j].Age
				}
			case "Name":
				if req.OrderBy > 0 {
					return result[i].Name < result[j].Name
				} else {
					return result[i].Name > result[j].Name
				}
			default:
				if req.OrderBy > 0 {
					return result[i].Id < result[j].Id
				} else {
					return result[i].Id > result[j].Id
				}
			}
		})
	}
	result = result[req.Offset:len(result)]
	if len(result) > req.Limit {
		result = result[0:req.Limit]
	}
	w.WriteHeader(http.StatusOK)
	usersMarshaled, _ := json.Marshal(result)
	w.Write(usersMarshaled)
}

func SearchServerBadJSON(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(badJSON))
}

func TestLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	client := SearchClient{
		AccessToken: "auth112",
		URL:         ts.URL,
	}
	req := SearchRequest{
		Limit: 10,
	}
	resp, err := client.FindUsers(req)
	if err != nil {
		t.Errorf("TestLimit err: " + err.Error())
	}
	if len(resp.Users) != 10 {
		t.Errorf("TestLimit incorrect len %v", len(resp.Users))
	}
}

func TestMaxLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	client := SearchClient{
		AccessToken: "auth112",
		URL:         ts.URL,
	}
	req := SearchRequest{
		Limit: 30,
	}
	resp, err := client.FindUsers(req)
	if err != nil {
		t.Errorf("TestMaxLimit err: " + err.Error())
	}
	if len(resp.Users) != 25 {
		t.Errorf("TestMaxLimit incorrect len %v", len(resp.Users))
	}
}

func TestNextPage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	client := SearchClient{
		AccessToken: "auth112",
		URL:         ts.URL,
	}
	req := SearchRequest{
		Limit: 25,
	}
	resp, err := client.FindUsers(req)
	if err != nil {
		t.Errorf("TestMaxLimit err: " + err.Error())
	}
	if !resp.NextPage {
		t.Errorf("TestMaxLimit: incorrect NextPage")
	}

	req.Offset = 20
	resp, err = client.FindUsers(req)
	if err != nil {
		t.Errorf("TestMaxLimit err: " + err.Error())
	}
	if resp.NextPage {
		t.Errorf("TestMaxLimit: incorrect NextPage")
	}
}

func TestBadRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServerErrors))
	client := SearchClient{
		URL: ts.URL,
	}
	req := SearchRequest{
		Offset: 2,
	}
	_, err := client.FindUsers(req)
	if err == nil || err.Error()[:24] != "cant unpack error json: " {
		t.Errorf("BadRequest unpack err json test failed")
	}

	req.Offset = 3
	_, err = client.FindUsers(req)
	if err == nil || err.Error()[:10] != "OrderFeld " {
		t.Errorf("BadRequest OrderFeld test failed")
	}

	req.Offset = 4
	_, err = client.FindUsers(req)
	if err == nil || err.Error()[:38] != "unknown bad request error: Other error" {
		t.Errorf("BadRequest unknown bad request error test failed")
	}
}

func TestNullURL(t *testing.T) {
	client := SearchClient{
		AccessToken: "auth112",
	}
	req := SearchRequest{
		Limit: 1,
	}
	_, err := client.FindUsers(req)
	if err == nil || err.Error()[:13] != "unknown error" {
		t.Errorf("Null URL incorrect test")
	}
}

func TestTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServerTimeout))
	client := SearchClient{
		AccessToken: "auth112",
		URL:         ts.URL,
	}
	req := SearchRequest{
		Limit: 1,
	}
	_, err := client.FindUsers(req)
	if err == nil || err.Error()[:12] != "timeout for " {
		t.Errorf("Timeout incorrect test")
	}
}

func TestAuth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	client := SearchClient{
		AccessToken: "auth115",
		URL:         ts.URL,
	}
	req := SearchRequest{
		Limit: 1,
	}
	_, err := client.FindUsers(req)
	if err == nil || err.Error() != "Bad AccessToken" {
		t.Errorf("Limit<0 incorrect test")
	}
}

func TestBadJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServerBadJSON))
	client := SearchClient{
		AccessToken: "auth115",
		URL:         ts.URL,
	}
	req := SearchRequest{
		Limit: 1,
	}
	_, err := client.FindUsers(req)
	if err == nil || err.Error()[:23] != "cant unpack result json" {
		t.Errorf("BadJSON bad test result")
	}
}

func TestInternalServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServerErrors))
	client := SearchClient{
		URL: ts.URL,
	}
	req := SearchRequest{
		Offset: 1,
	}
	_, err := client.FindUsers(req)
	if err == nil || err.Error() != "SearchServer fatal error" {
		t.Errorf("InternalServerError test failed")
	}
}

func TestBadLimit(t *testing.T) {
	client := SearchClient{
		AccessToken: "",
		URL:         "",
	}
	req := SearchRequest{
		Limit: -1,
	}
	_, err := client.FindUsers(req)
	if err == nil || err.Error() != "limit must be > 0" {
		t.Errorf("Limit<0 incorrect test")
	}
}

func TestOffset(t *testing.T) {
	client := SearchClient{
		AccessToken: "",
		URL:         "",
	}
	req := SearchRequest{
		Offset: -1,
	}
	_, err := client.FindUsers(req)
	if err == nil || err.Error() != "offset must be > 0" {
		t.Errorf("Offset<0 incorrect test")
	}
}
