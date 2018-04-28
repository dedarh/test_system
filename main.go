package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"text/template"

	"github.com/dedarh/test_system/templates"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

var (
	store *sessions.CookieStore
)
var config struct {
	DbLogin        string `json:"dbUser"`
	DbHost         string `json:"dbHost"`
	DbName         string `json:"dbDb"`
	DbPassword     string `json:"dbPassword"`
	PathToConfVm   string `json:"PathToConfVm"`
	PathTestResult string `json:"PathTestResult"`
	CookieName     string `json:"CookieName"`
	Key            string `json:"Key"`
	Salt           string `json:"Salt"`
}

var configFile = flag.String("config", "conf.json", "Where to read the config from")

type server struct {
	Db *sqlx.DB
}
type VmVariables struct {
	Password string
	Box      string
	Hostname string
	Memory   string
	Login    string
	Port     int
}
type User struct {
	Id         int    `db:"id_user"`
	Login      string `db:"login"`
	Password   string `db:"password"`
	Name       string `db:"name"`
	Gr         int    `db:"gr"`
	Token      string `db:"token"`
	Permission string `db:"permission"`
	Firstname  string `db:"firstname"`
	Lastname   string `db:"lastname"`
	State      int    `db:"state"`
}
type Test struct {
	Id   int    `db:"i_test"`
	Name string `db:"name"`
}
type TestQuestion struct {
	IdQuestion string `db:"i_question"`
	Text       string `db:"question_name"`
	Type       string `db:"type"`
	Answer     []TestQuestionAnswer
}
type TestQuestionAnswer struct {
	IdQuestion int    `db:"i_question"`
	IdAnswer   int    `db:"i_answer"`
	Text       string `db:"text"`
	Correct    int    `db:"status"`
}

func loadConfig() error {
	jsonData, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return err
	}
	store = sessions.NewCookieStore([]byte(config.Key))
	return json.Unmarshal(jsonData, &config)
}
func (s *server) getUserFromDbByLogin(login string) (user User, err error) {
	err = s.Db.Get(&user, "SELECT * FROM users WHERE users.login= $1", login)
	return
}

func getSha(str string) string {
	bytes := []byte(str)
	h := sha256.New()
	h.Write(bytes)
	code := h.Sum(nil)
	codeStr := hex.EncodeToString(code)
	return codeStr
}

func (s *server) MakeVagrantConf(login string) error {
	usr, err := s.getUserFromDbByLogin(login)
	if err != nil {
		return err
	}
	generatedVm := VmVariables{Box: "ubuntu/xenial64", Hostname: "testvm" + login + "", Memory: "2048", Port: usr.Id}
	templates, err := template.ParseFiles("VagrantConfSample.txt")
	if err != nil {
		return err
	}
	if _, err := os.Stat(config.PathToConfVm + string(usr.Login) + "/" + usr.Login + "/"); os.IsNotExist(err) {
		os.Mkdir(config.PathToConfVm+string(usr.Login)+"/", 0777)
		os.Mkdir(config.PathToConfVm+string(usr.Login)+"/"+usr.Login+"/", 0777)
	}
	file, err := os.Create(config.PathToConfVm + string(usr.Login) + "/" + usr.Login + "/Vagrantfile")
	if err != nil {
		log.Println(err)
	}
	defer file.Close()
	templates.ExecuteTemplate(file, "VagrantConfSample.txt", generatedVm)

	errFile := s.MakeBat(login)
	if errFile != nil {
		log.Println(err)
	}
	return err
}
func (s *server) MakeBat(login string) error {
	usr, err := s.getUserFromDbByLogin(login)
	text := "vagrant up"
	file, err := os.Create(config.PathToConfVm + string(usr.Login) + "/" + usr.Login + "/Vagrantfile.bat")
	file.WriteString(text)
	defer file.Close()
	text2 := "vagrant halt"
	file2, err := os.Create(config.PathToConfVm + string(usr.Login) + "/" + usr.Login + "/Vagranthalt.bat")
	file2.WriteString(text2)
	defer file2.Close()
	return err
}
func (s *server) ExecuteVagrant(path string) error {
	log.Println(path)
	vagrant := exec.Command(path + "Vagrantfile.bat")
	vagrant.Dir = path
	err := vagrant.Run()
	return err
}
func (s *server) ExecuteVagrantHalt(path string) error {
	vagrant := exec.Command(path + "Vagranthalt.bat")
	vagrant.Dir = path
	err := vagrant.Run()
	return err
}
func (s *server) CreatedVds(w http.ResponseWriter, r *http.Request) {
	err := s.MakeVagrantConf(r.FormValue("user"))
	if err != nil {
		log.Println(err)
		return
	}

	if err = s.ExecuteVagrant(config.PathToConfVm + r.FormValue("user") + "/" + r.FormValue("user") + "/"); err != nil {
		log.Println(err)
		return
	}

	if _, err := s.Db.Exec("INSERT INTO suggestions (id, login, state, status) VALUES ((SELECT ifnull(max(id), 0)+1 FROM suggestions),'" + r.FormValue("user") + "', 0,0)"); err != nil {
		log.Print(errors.New("Ошибка добавление в базу"))
		return
	}
}
func (s *server) StopdVds(w http.ResponseWriter, r *http.Request) {
	if err := s.ExecuteVagrantHalt(config.PathToConfVm + r.FormValue("user") + "/" + r.FormValue("user") + "/"); err != nil {
		log.Println(err)
		return
	}
}
func (s *server) UsersIndex(w http.ResponseWriter, r *http.Request) { //игформация о пользователе
	var users []User
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}
	if err := s.Db.Select(&users, "SELECT * FROM users ORDER BY Name ASC"); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	answerUser, err := json.Marshal(users)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprint(w, string(answerUser))
}
func (s *server) UsersShow(w http.ResponseWriter, r *http.Request) { //игформация об определенном пользователе
	var user User
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	if err := s.Db.Get(&user, "SELECT * FROM users WHERE id_user  = $1", r.FormValue("id")); err != nil {
		http.NotFound(w, r)
		return
	}

	answerUser, err := json.Marshal(user)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprint(w, string(answerUser))
}
func (s *server) TestIndex(w http.ResponseWriter, r *http.Request) { //все тесты
	var test []Test
	if r.Method != "POST" {
		http.Error(w, http.StatusText(405), 405)
		return
	}
	if err := s.Db.Select(&test, "SELECT * FROM test_name  ORDER BY name ASC"); err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}
	answerTest, err := json.Marshal(test) //govnocode
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintf(w, string(answerTest))
}
func (s *server) TestStart(w http.ResponseWriter, r *http.Request) { //создание теста и отправка его пользователю
	if r.Method == "GET" {
		id := r.FormValue("id")
		fmt.Fprint(w, templates.TestsPageView(id))
	}
	if r.Method == "POST" {
		var TestQuestion []TestQuestion
		id := r.FormValue("id") //id теста
		err := s.Db.Select(&TestQuestion, `SELECT q.question_name,t_q.i_question, q.type FROM "test_question" t_q JOIN "question" q ON t_q.i_question = q.i_question  WHERE t_q.i_test = $1 ORDER BY q.i_question DESC`, id)
		if err != nil {
			http.Error(w, http.StatusText(500), 500)
			fmt.Println(err)
			return
		}
		for index, e := range TestQuestion {
			var tqa []TestQuestionAnswer
			s.Db.Select(&tqa, "SELECT i_question, i_answer, text FROM answer WHERE i_question = $1", e.IdQuestion)
			TestQuestion[index].Answer = tqa
		}
		answerTest, err := json.Marshal(TestQuestion)
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Fprintf(w, string(answerTest))
	}

}
func (s *server) TestCheckQuestionWith(w http.ResponseWriter, r *http.Request) { //проверка верного ответа на вопрос  http://localhost:4080/test_check_qestion/?Id_Test=1&Answer_user=[{"Id_Question":1,"ID_Answer":1,"Text":"test%20text"},{"Id_Question":1,"ID_Answer":2,"Text":"test%20text"},{"Id_Question":1,"ID_Answer":3,"Text":"test%20text"},{"Id_Question":2,"ID_Answer":5,"Text":"%20"}]
	session, _ := store.Get(r, config.CookieName)
	var AnswerUserJson []TestQuestionAnswer
	var idUser = strconv.Itoa(session.Values["id"].(int))
	var IdFile = ""
	if r.Method != "POST" {
		http.Error(w, http.StatusText(405), 405)
		return
	}
	userAnswer := []byte(r.FormValue("Answer_user"))
	err := json.Unmarshal(userAnswer, &AnswerUserJson)
	if err != nil {
		log.Println(err)
		return
	}
	if err := s.Db.Get(&IdFile, "SELECT ifnull(COUNT(id)+1, 0) FROM user_answer"); err != nil {
		log.Fatal(err)
	}
	os.Mkdir(config.PathTestResult+"user_"+idUser+"/", 0777)
	os.Mkdir(config.PathTestResult+"user_"+idUser+"/id_test_"+r.FormValue("Id_Test")+"/", 0777)
	file, err := os.Create(config.PathTestResult + "user_" + idUser + "/id_test_" + r.FormValue("Id_Test") + "/answer_" + IdFile + "_with.json")
	file.Write(userAnswer)
	file.Close()
}
func (s *server) TestCheckQuestion(w http.ResponseWriter, r *http.Request) { //проверка верного ответа на вопрос  http://localhost:4080/test_check_qestion/?Id_Test=1&Answer_user=[{"Id_Question":1,"ID_Answer":1,"Text":"test%20text"},{"Id_Question":1,"ID_Answer":2,"Text":"test%20text"},{"Id_Question":1,"ID_Answer":3,"Text":"test%20text"},{"Id_Question":2,"ID_Answer":5,"Text":"%20"}]
	session, _ := store.Get(r, config.CookieName)
	var countCorrect = 0
	var countWrong = 0
	var QuestionAnswerCorrect []TestQuestionAnswer
	var AnswerUserJson []TestQuestionAnswer
	var IdFile = ""
	if r.Method != "POST" {
		http.Error(w, http.StatusText(405), 405)
		return
	}
	userAnswer := []byte(r.FormValue("Answer_user"))
	err := json.Unmarshal(userAnswer, &AnswerUserJson)
	if err != nil {
		log.Println(err)
		return
	}
	for index, e := range AnswerUserJson {
		if err := s.Db.Select(&QuestionAnswerCorrect, `SELECT i_question, i_answer, text, status FROM answer WHERE i_question = $1 AND i_answer = $2`, e.IdQuestion, e.IdAnswer); err != nil {
			http.Error(w, http.StatusText(500), 500)
			fmt.Println(err)
			return
		}
		if QuestionAnswerCorrect[index].Correct == 1 {
			//log.Println("верный овтет")
			countCorrect++
		} else {
			//log.Println("не верный овтет")
			countWrong++
		}
	}
	var returnObj = map[string]int{"countCorrect": countCorrect, "countWrong": countWrong}
	returnObjJson, _ := json.Marshal(returnObj)
	var idUser = strconv.Itoa(session.Values["id"].(int))
	if _, err := s.Db.Exec("INSERT INTO user_answer (id, id_user, id_test, answer_temp) VALUES ((SELECT ifnull(max(id), 0)+1 FROM user_answer), '" + idUser + "' , '" + r.FormValue("Id_Test") + "','" + string(returnObjJson) + "')"); err != nil {
		log.Print("Ошибка добавление в базу")
		http.Error(w, http.StatusText(500), 500)
		return
	}
	if err := s.Db.Get(&IdFile, "SELECT ifnull(COUNT(id), 0) FROM user_answer"); err != nil {
		log.Fatal(err)
	}

	os.Mkdir(config.PathTestResult+"user_"+idUser+"/", 0777)
	os.Mkdir(config.PathTestResult+"user_"+idUser+"/id_test_"+r.FormValue("Id_Test")+"/", 0777)
	file, err := os.Create(config.PathTestResult + "user_" + idUser + "/id_test_" + r.FormValue("Id_Test") + "/answer_" + IdFile + "_without.json")

	file.Write(userAnswer)
	file.Close()
	fmt.Fprintf(w, string(returnObjJson))
}
func (s *server) Index(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, config.CookieName)
	// проверка на вторизацию
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		//http.Error(w, "Не авторизованный", http.StatusForbidden)
		http.Redirect(w, r, "/login/", 302)
		return
	}
	fmt.Fprint(w, templates.TestsPage())
	//fmt.Fprintln(w, "Авторизованный")
}
func (s *server) login(w http.ResponseWriter, r *http.Request) { //http://localhost:4080/login/?password="dedarh"&login="dedarh"
	session, _ := store.Get(r, config.CookieName)
	if r.Method == "GET" {
		fmt.Fprint(w, templates.LoginPage())
	}
	if r.Method == "POST" {

		//(sha256sha256(password + salt))

		sha := getSha(r.FormValue("password"));
		shaHash := getSha(sha + config.Salt);
		log.Println(shaHash)
		var user User
		if err := s.Db.Get(&user, "SELECT * FROM users WHERE password  = $1 and login = $2 LIMIT 1", shaHash, r.FormValue("login")); err != nil {
			log.Print(err)
			return
		}

		session.Values["authenticated"] = true
		session.Values["id"] = user.Id
		session.Values["names"] = user.Firstname
		session.Values["lastname"] = user.Lastname
		session.Values["admin"] = user.Permission
		session.Values["group"] = user.Gr
		session.Save(r, w)
		http.Redirect(w, r, "/secret_test/", 302)
	}

}
func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, config.CookieName)
	session.Values["authenticated"] = false
	session.Save(r, w)
	http.Redirect(w, r, "/login/", 302)
}

func main() {
	if err := loadConfig(); err != nil {
		panic(err)
	}
	log.Println("Config loaded from", *configFile)

	s := server{Db: sqlx.MustConnect("postgres", "postgresql://"+config.DbLogin+":"+config.DbPassword+"@"+config.DbHost+":26257/"+config.DbName+"?sslmode=disable")}
	defer s.Db.Close()

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/users/", s.UsersIndex)
	http.HandleFunc("/users/show/", s.UsersShow)
	http.HandleFunc("/getTest/", s.TestIndex)
	http.HandleFunc("/testStart/", s.TestStart)
	http.HandleFunc("/testCheckQuestion/", s.TestCheckQuestion)
	http.HandleFunc("/testCheckQuestionWhith/", s.TestCheckQuestionWith)

	http.HandleFunc("/logout/", s.logout)
	http.HandleFunc("/login/", s.login)

	http.HandleFunc("/startVds/", s.CreatedVds)
	http.HandleFunc("/stopVds/", s.StopdVds)
	http.HandleFunc("/", s.Index)

	http.ListenAndServe(":4080", nil)
}
