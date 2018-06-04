package main

import (
	"bufio"
	"bytes"
	"fmt"
//	"io"
	"encoding/json"
	"encoding/csv"
//	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
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

type cbase map[string]string

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
	b := base{}
	update(b)
	b = b.rotate()
	updateGVM(b)
	b.render()
}

func update(b base) {
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
	marshalResponse(maininfo, &missionaries)

	var assign []map[string]interface{}
	marshalResponse(assignments, &assign)
	for _, i := range missionaries {
		b.unmarshalMain(i)
	}
	for _, i := range assign {
		b.unmarshalAssign(i)
	}
	for k, _ := range b {
		unitres, _ := client.Get("https://imos.ldschurch.org/ws/profile-contacts/Services/rest/14361/org/missionary/" + strconv.Itoa(int(k)) + "?cachebust=20548")
		var unit map[string]interface{}
		marshalResponse(unitres, &unit)
		b.unmarshalUnit(unit, k)
	}
}

func (b base) render() {
	dbf, _ := os.Create("db.csv")
	db := csv.NewWriter(dbf)
	keys := []string{"legacyMissId", "missType", "lastName", "firstName",
				"email", "area", "district", "zone", "phoneNumber", "mtcStartDate",
				"assignmentStart", "assignmentEnd", "language",
				"unitTypeName", "unitName", "unitTitleDescription",
				"unitLeaderName", "unitHomePhone", "unitCellPhone",
				"unitEmailAddress", "parentUnitTypeName", "parentUnitName",
				"parentUnitTitleDescription", "parentUnitLeaderName", "parentUnitHomePhone",
				"parentUnitCellPhone", "parentUnitEmailAddress", "citizenship", "Passport",
				"PassportIssued", "PassportExpires", "Carne", "CarneIssued", "CarneExpires"}
	db.Write(keys)
	for _, rec := range b {
		var buf []string
		for _, v := range keys {
			switch rec[v].(type) {
			case string:
				buf = append(buf, rec[v].(string))
			case float64:
				buf = append(buf, strconv.Itoa(int(rec[v].(float64))))
			}
		}
		db.Write(buf)
	}
}

func (b base) unmarshalMain(m map[string]interface{}) {
	id := m["missionaryId"].(float64)
	b[id] = mapjoin(b[id], mapflatten(m))
}

func (b base) unmarshalAssign(m map[string]interface{}) {
	id := m["missionaryId"].(float64)
	if m["missionId"].(float64) == 14361 {
		m["assignmentEnd"] = dateString(m["assignmentEnd"].(float64))
		m["assignmentStart"] = dateString(m["assignmentStart"].(float64))
	}
	if _, ok := m["mtcStartDate"].(float64); ok {
		m["mtcStartDate"] = dateString(m["mtcStartDate"].(float64))
	}
	b[id] = mapjoin(b[id], m)
}
	
func (b base) unmarshalUnit(m map[string]interface{}, id float64) {
	n := m["parentUnit"].(map[string]interface{})
	o := m["unit"].(map[string]interface{})
	res := make(map[string]interface{})
	delete(n, "unitNumber")
	delete(o, "unitNumber")
	delete(n, "parentUnitNumber")
	delete(o, "parentUnitNumber")
	for k, v := range n {
		if k == "leader" {
			delete(v.(map[string]interface{}), "address")
			for l, u := range v.(map[string]interface{}) {
				res["parentUnit"+ strings.Title(l)] = u
			}
		} else {
			res["parent"+ strings.Title(k)] = v
		}
	}
	for k, v := range o {
		if k == "leader" {
			delete(v.(map[string]interface{}), "address")
			for l, u := range v.(map[string]interface{}) {
				res["unit"+ strings.Title(l)] = u
			}
		} else {
			res[k] = v
		}
	}
	b[id] = mapjoin(b[id], res)
}

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
	return m
}

func updateGVM(b base) {
	data := url.Values{}
	data.Set("j_submit", "Sign In")
	ldataf, _ := os.Open("gvmlogin")
	ldata := bufio.NewScanner(ldataf)
	ldata.Scan()
	uname := ldata.Text()
	ldata.Scan()
	pword := ldata.Text()
	
	data.Set("j_username", uname)
	data.Set("j_password", pword)
	jar := newJar()


	client := &http.Client{
		Jar: jar,
	}

	client.Get("https://apps.lds.org/gvm/security/login.jsf")
	
	req, _ := http.NewRequest("POST", "https://apps.lds.org/gvm/security/j_acegi_security_check", strings.NewReader(data.Encode()))
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.56 Safari/537.36")

	client.Do(req)
	

	res, _ := client.Get("https://apps.lds.org/gvm/missionary/roster.jsf")
	
	

	doc, _ := goquery.NewDocumentFromReader(res.Body)
	num, _ := doc.Find(".pagination tr td select option").Last().Attr("value")
	javatag, _ := doc.Find(".yui-t7 div div form input").Eq(1).Attr("value")
	
	data = url.Values{}
	data.Set("j_id61", "j_id61")
	data.Set("j_id61:j_id120", num)
	data.Set("javax.faces.ViewState", javatag)
	
	req, _ = http.NewRequest("POST", "https://apps.lds.org/gvm/missionary/roster.jsf", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.56 Safari/537.36")

	res, _ = client.Do(req)

	doc, _ = goquery.NewDocumentFromReader(res.Body)
	

	c := unmarshallCitizen()
	table := doc.Find(".table-data tbody tr")
	unmarsh := makeunmarshalgvm(b, c)
	table.Each(unmarsh)
	
	fmt.Println(b)
}

func dateString(u float64) string {
	return time.Unix(int64(u/1000), 0).Format("02/01/2006")
}

func (b base) rotate() base {
	c := base{}
	for _, rec := range b {
		c[rec["legacyMissId"].(float64)] = rec
	}
	return c
}
func makeunmarshalgvm(b base, c cbase) func(int, *goquery.Selection) {

unmarshalgvm := func (_ int, s *goquery.Selection) {
	var id float64
	m := record{}
	docs := map[string]int{}

	parseDocs := func(i int, t *goquery.Selection)  {
		s := strings.TrimSpace(t.Text())
		switch s {
		case "Pasaporte", "Passport":
			if _, ok := docs["Passport"]; !ok {
				docs["Passport"] = i
			}
		case "Carne de Extranjeria":
			if _, ok := docs["Carne"]; !ok {
				docs["Carne"] = i
			}
		}
	}
	
	unmarshalgvmrecord := func (i int, t *goquery.Selection) {
		switch i {
		case 0:
			s :=strings.TrimSpace(t.Find(".travelerinfo").Text())
			x, _ := strconv.Atoi(s[14:len(s)-1])
			id = float64(x)
		case 1:
			s := strings.TrimSpace(t.First().Text())
			if st, ok := c[s]; ok {
				s = st
			}
			m["citizenship"] = s
		case 3:
			t.Find("div").Each(parseDocs)
		case 5:
			for d, ind := range docs {
				m[d] = strings.TrimSpace(t.Find("div").Eq(ind).Text())
			}
			if _, ok := m["Passport"]; !ok {
				m["Passport"] = ""
			}
			if _, ok := m["Carne"]; !ok {
				m["Carne"] = ""
			}
		case 6:
			for d, ind := range docs {
				str := strings.TrimSpace(t.Find("div").Eq(ind).Text())
				dat, _ := time.Parse("2 Jan 2006", str)
				m[d+"Issued"] = dat.Format("02/01/2006")
			}
			if _, ok := m["PassportIssued"]; !ok {
				m["PassportIssued"] = ""
			}
			if _, ok := m["CarneIssued"]; !ok {
				m["CarneIssued"] = ""
			}
		case 7:
			for d, ind := range docs {
				str := strings.TrimSpace(t.Find("div").Eq(ind).Text())
				dat, _ := time.Parse("2 Jan 2006", str)
				m[d+"Expires"] = dat.Format("02/01/2006")
			}
			if _, ok := m["PassportExpires"]; !ok {
				m["PassportExpires"] = ""
			}
			if _, ok := m["CarneExpires"]; !ok {
				m["CarneExpires"] = ""
			}
		}
	}
	
	s.Find("td").Each(unmarshalgvmrecord)
	b[id] = mapjoin(b[id], m)
}
return unmarshalgvm
}

func unmarshallCitizen() map[string]string {
	base := map[string]string{}
	citf, _ := os.Open("cit.csv")
	cit := csv.NewReader(citf)
	cits, _ := cit.ReadAll()
	for _, v := range cits {
		base[v[0]] = v[1]
	}
	return base
}

func (b cbase) Add(country, trans string) {
	b[country] = trans
}

func (b cbase) render() {
	citf, _ := os.Create("cit.csv")
	cit := csv.NewWriter(citf)
	for k, v := range b {
		cit.Write([]string{k, v})
	}
}