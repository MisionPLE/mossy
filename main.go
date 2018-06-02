package main

import (
	"bufio"
	"bytes"
	"fmt"
//	"io"
	"encoding/json"
//	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)
type myjar struct {
	j []*http.Cookie
	m map[string]int
}

func newJar() *myjar {
	var j myjar
	j.m = make(map[string]int)
	return &j
}

func updateCookies(m map[string]int, ncs []*http.Cookie, ocs *[]*http.Cookie) {
	var ok bool
	var i int
	for _, c := range ncs {
		if i, ok = m[c.Name]; ok {
			(*ocs)[i] = c
		} else {
			m[c.Name] = len(*ocs)
			*ocs = append(*ocs, c)
		}
	}
}

func (j *myjar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	updateCookies(j.m, cookies, &j.j)
}

func (j myjar) Cookies(u *url.URL) []*http.Cookie {
	return j.j
}

type record map[string]interface{}


type base map[float64]record

type ltime struct {
	time.Time
}

func (t ltime) String() string {
	return t.Format("02/01/2006")
}

func parseTime(f float64) ltime {
	return ltime{time.Unix(int64(f/1000),0)}
}

func main() {
	data := url.Values{}
	data.Set("goto", "")
	data.Set("gotoOnFail", "")
	data.Set("SunQueryParamsString", "c2VydmljZT1jcmVkZW50aWFscw==")
	data.Set("IDButton", "Log In")
	data.Set("encoded", "false")
	data.Set("gx_charset", "UTF-8")
	ldataf, err := os.Open("login")
	ldata := bufio.NewScanner(ldataf)
	ldata.Scan()
	uname := ldata.Text()
	ldata.Scan()
	pword := ldata.Text()
	
	data.Set("IDToken1", uname)
	data.Set("IDToken2", pword)
	jar := newJar()


	client := &http.Client{
		Jar: jar,
	}

	client.Get("https://ident.lds.org/sso/UI/Login?service=credentials")

	req, _ := http.NewRequest("POST", "https://ident.lds.org/sso/UI/Login", strings.NewReader(data.Encode()))
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.56 Safari/537.36")

	client.Do(req)

	maininfo, err := client.Get("https://imos.ldschurch.org/ws/roster/Services/rest/14361/missionaries?cachebust=56020")

	assignments, err := client.Get("https://imos.ldschurch.org/ws/roster-profile/14361/assignments?cachebust=90011")

	buffer := new(bytes.Buffer)
	marshalResponse := func(res *http.Response, f interface{}) error {
		buffer.ReadFrom(res.Body)
		err = json.Unmarshal(buffer.Bytes(), &f)
		res.Body.Close()
		buffer.Reset()
		return err
	}

	var missionaries []map[string]interface{}
	err = marshalResponse(maininfo, &missionaries)

	var assign []map[string]interface{}
	err = marshalResponse(assignments, &assign)
	keys := []string{"legacyMissId", "missionaryId", "missType", "lastName", "firstName",
				"email", "area", "district", "zone", "phoneNumber", "mtcStartDate",
				"assignmentStart", "assignmentEnd", "language"}

	for k, v := range missionaries[0] {
		if w, ok := v.(map[string]interface{}); ok {
			for l, _ := range w {
				keys = append(keys, l)
			}
		} else {
			keys = append(keys, k)
		}
	}

	fmt.Println(keys)

	for _, i := range missionaries {
		fmt.Println(i["missionaryId"].(float64), i["email"].(string))
	}
	
//	_, err = io.Copy(os.Stdout, maininfo.Body)
//	_, err = io.Copy(os.Stdout, assignments.Body)

}

func (b base) unmarshalMain(m map[string]interface{}) {
	id := m["missionaryId"].(float64)
	b[id] = mapjoin(b[id], mapflatten(m))
}

func (b base) unmarshalAssign(m map[string]interface{}) {
	id := m["missionaryId"].(float64)
	m["assignmentEnd"] = parseTime(m["assignmentEnd"].(float64))
	m["assignmentStart"] = parseTime(m["assignmentStart"].(float64))
	m["mtcStartDate"] = parseTime(m["mtcStartDate"].(float64))
	b[id] = mapjoin(b[id], m)
}
	
func unmarshalContact(m map[string]interface{}) record {
	
func mapjoin(m, n map[string]interface{}) map[string]interface{} {
	if m == nil {
		m = make(map[string]interface{})
	}
	for k, v := range n {
		m[k] = v
	}
	return m
}

func mapflatten(m map[string]interface{}) map[string]interface{} {
	for k, v := range m {
		if w, ok := v.(map[string]interface{}); ok {
			for l, x := range w {
				m[l] = x
			}
			delete(m, k)
		}
	}
}

func dateString(u float64) string {
	return time.Unix(int64(u/1000)).Format("02/01/2006")
}