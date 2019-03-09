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

	"./templates"
	//"github.com/dedarh/test_system/templates"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var (
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	key   = []byte(config.Key)
	store = sessions.NewCookieStore(key)
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
var db *sqlx.DB
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
	Permission string `db:"permission"`
	Firstname  string `db:"firstname"`
	Lastname   string `db:"lastname"`
}
type (
	Test = templates.Test
	TestAdmin = templates.TestAdmin
	TestQuestionAnswer = templates.TestQuestionAnswer
	TestQuestionAdmin = templates.TestQuestionAdmin
)

type TestAnswerAdmin struct {
	Id         string `db:"id"`
	IdUser     string `db:"id_user"`
	IdTest     string `db:"id_test"`
	AnswerTemp string `db:"answer_temp"`
	State      string `db:"state"`
}
type TestQuestion struct {
	IdQuestion string `db:"i_question"`
	Text       string `db:"question_name"`
	Type       string `db:"type"`
	Answer     []TestQuestionAnswer
}


func loadConfig() error {
	jsonData, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, &config)
}
func (s *server) getUserFromDbByLogin(login string) (User, error) {
	var users User
	err := s.Db.Get(&users, "SELECT * FROM users WHERE users.login= $1", login)
	return users, err
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
	var login = r.FormValue("user")
	err := s.MakeVagrantConf(login)
	s.ExecuteVagrant(config.PathToConfVm + login + "/" + login + "/")
	if err != nil {
		log.Println(err)
		return
	}
	if _, err := s.Db.Exec("INSERT INTO suggestions (id, login, state, status) VALUES ((SELECT ifnull(max(id), 0)+1 FROM suggestions),'" + login + "', 0,0)"); err != nil {
		log.Print("Ошибка добавление в базу")
		return
	}
}
func (s *server) StopdVds(w http.ResponseWriter, r *http.Request) {
	var login = r.FormValue("user")
	error := s.ExecuteVagrantHalt(config.PathToConfVm + login + "/" + login + "/")
	if error != nil {
		log.Println(error)
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

	for _, u := range users {
		answerUser, err := json.Marshal(u) //govnocode
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Fprintf(w, string(answerUser))
	}
}
func (s *server) UsersShow(w http.ResponseWriter, r *http.Request) { //игформация об определенном пользователе
	var user []User
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	if err := s.Db.Get(&user, "SELECT * FROM users WHERE id_user  = $1", r.FormValue("id")); err != nil {
		http.NotFound(w, r)
		return
	}

	answerUser, err := json.Marshal(user[0]) //govnocode
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintf(w, string(answerUser))
}
func (s *server) TestStart(w http.ResponseWriter, r *http.Request) { //создание теста и отправка его пользователю
	if r.Method == "GET" {
		id := r.FormValue("id")
		fmt.Fprint(w, templates.TestsPageView(id))
	}
	if r.Method == "POST" {
		var TestQuestion []TestQuestion
		id := r.FormValue("id") //id теста
		err := s.Db.Select(&TestQuestion, `SELECT q.question_name,t_q.i_question, q.type FROM "test_question" t_q JOIN "question" q ON t_q.i_question = q.i_question  WHERE t_q.i_test = $1 ORDER BY q.i_question ASC `, id)
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
	if err := s.Db.Get(&IdFile, "SELECT MAX(id + 1) FROM user_answer"); err != nil {
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
	var countCorrect = 0.0
	var countWrong = 0.0
	var QuestionAnswerCorrect []TestQuestionAnswer
	var AnswerUserJson []TestQuestionAnswer
	var IdFile = ""
	var state = ""
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
		if err := s.Db.Select(&QuestionAnswerCorrect, `SELECT i_question, i_answer, text, correct FROM answer WHERE i_question = $1 AND i_answer = $2`, e.IdQuestion, e.IdAnswer); err != nil {
			http.Error(w, http.StatusText(500), 500)
			fmt.Println(err)
			return
		}
		if QuestionAnswerCorrect[index].Correct == 1 {
			countCorrect++
			AnswerUserJson[index].Correct = 1
		} else {
			countWrong++
			AnswerUserJson[index].Correct = 0
		}
	}
	if (countCorrect*0.5 > countWrong) {
		state = "Completed test"
	} else {
		state = "Failed test"
	}
	var returnObj = map[string]float64{"countCorrect": countCorrect, "countWrong": countWrong}
	returnObjJson, _ := json.Marshal(returnObj)
	var idUser = strconv.Itoa(session.Values["id"].(int))
	if _, err := s.Db.Exec("INSERT INTO user_answer (id, id_user, id_test, answer_temp, state) VALUES ((SELECT ifnull(max(id), 0)+1 FROM user_answer), '" + idUser + "' , '" + r.FormValue("Id_Test") + "','" + string(returnObjJson) + "','" + state + "')"); err != nil {
		log.Print("Ошибка добавление в базу")
		http.Error(w, http.StatusText(500), 500)
		return
	}
	if err := s.Db.Get(&IdFile, "SELECT MAX(id) FROM user_answer "); err != nil {
		log.Fatal(err)
	}

	os.Mkdir(config.PathTestResult+"user_"+idUser+"/", 0777)
	os.Mkdir(config.PathTestResult+"user_"+idUser+"/id_test_"+r.FormValue("Id_Test")+"/", 0777)
	file, err := os.Create(config.PathTestResult + "user_" + idUser + "/id_test_" + r.FormValue("Id_Test") + "/answer_" + IdFile + "_without.json")

	out, err := json.Marshal(AnswerUserJson)
	if err != nil {
		log.Print(err)
	}

	file.Write(out)
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
	var test []Test
	if err := s.Db.Select(&test, "SELECT * FROM test_name  ORDER BY name ASC"); err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	fmt.Fprint(w, templates.TestsPage(test))
	//fmt.Fprintln(w, "Авторизованный")
}

func (s *server) login(w http.ResponseWriter, r *http.Request) { //http://localhost:4080/login/?password="dedarh"&login="dedarh"
	session, _ := store.Get(r, config.CookieName)
	if r.Method == "GET" {
		fmt.Fprint(w, templates.LoginPage())
	}
	if r.Method == "POST" {

		//(sha256sha256(password + salt))

		sha := getSha(r.FormValue("password") + config.Salt);
		shaHash := getSha(sha);
		fmt.Println(shaHash);
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

func (s *server) IndexAdmin(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, config.CookieName)
	// проверка на вторизацию
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		//http.Error(w, "Не авторизованный", http.StatusForbidden)
		http.Redirect(w, r, "/login/", 302)
		return
	}
	if (session.Values["admin"] != "admin") {
		http.Redirect(w, r, "/", 302)
		return
	}
	var test []TestAdmin
	if err := s.Db.Select(&test, `SELECT atemp.id, t.i_test, t.name,u.firstname, u.lastname, atemp.state FROM "user_answer" atemp JOIN "users" u ON  atemp.id_user = u.id_user JOIN "test_name" t ON atemp.id_test = t.i_test WHERE atemp.state != 'NULL' ORDER BY atemp.id DESC `); err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	log.Println(test)
	fmt.Fprint(w, templates.AdminPage(test))
	//fmt.Fprintln(w, "Авторизованный")
}
func (s *server) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		session, _ := store.Get(r, config.CookieName)
		// проверка на вторизацию
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			//http.Error(w, "Не авторизованный", http.StatusForbidden)
			fmt.Println("Не авторизованный")
			//http.Redirect(w, r, "/login/", 302)
			return
		}
		if (session.Values["admin"] != "admin") {
			http.Redirect(w, r, "/", 302)
			return
		}
		fmt.Fprint(w, templates.CreateQuestionPage())
		//fmt.Fprintln(w, "Авторизованный")
	}
	if r.Method == "POST" {
		session, _ := store.Get(r, config.CookieName)
		// проверка на вторизацию
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			//http.Error(w, "Не авторизованный", http.StatusForbidden)
			fmt.Println("Не авторизованный")
			//http.Redirect(w, r, "/login/", 302)
			return
		}
		if (session.Values["admin"] != "admin") {
			http.Redirect(w, r, "/", 302)
			return
		}

		createQuestionType:= []byte(r.FormValue("type_q"))
		createQuestion := string([]byte(r.FormValue("quetion"))[:])



		if(string(createQuestionType) == "0"){
			log.Print("Создание текстового вопроса")
			log.Print(createQuestion)
			if _, err := s.Db.Exec("INSERT INTO question (i_question, question_name, type) VALUES ((SELECT ifnull(max(i_question), 0)+1 FROM question),'" + string(createQuestion) +"','0')"); err != nil {
				log.Print("Ошибка добавление в базу")
			}

		}
		if(string(createQuestionType) == "1"){
			log.Print("Создание Cheackboks вопроса")
			if _, err := s.Db.Exec("INSERT INTO question (i_question, question_name, type) VALUES ((SELECT ifnull(max(i_question), 0)+1 FROM question),'" + string(createQuestion) +"','1')"); err != nil {
				log.Print("Ошибка добавление в базу")
			}
			var IdQuest = ""
			if err := s.Db.Get(&IdQuest, "SELECT MAX(i_question) FROM question "); err != nil {
				log.Fatal(err)
			}


/*
type1 := string([]byte(r.FormValue("answer1t"))[:])
			type2 := string([]byte(r.FormValue("answer2t"))[:])
			type3 := string([]byte(r.FormValue("answer3t"))[:])
			type4 := string([]byte(r.FormValue("answer4t"))[:])

			answer1 := string([]byte(r.FormValue("answer1"))[:])
			answer2 := string([]byte(r.FormValue("answer2"))[:])
			answer3 := string([]byte(r.FormValue("answer3"))[:])
			answer4 := string([]byte(r.FormValue("answer4"))[:])
			log.Print(type1)
			log.Print(type2)
			log.Print(type3)
			log.Print(type4)



			if(answer1 != ""){
				if _, err := s.Db.Exec("INSERT INTO answer (i_answer, i_question, correct, text) VALUES ((SELECT ifnull(max(i_answer), 0)+1 FROM answer),'" + string(IdQuest) +"','"+type1+"','"+answer1+"')"); err != nil {
					log.Print("Ошибка добавление в базу")
				}
			}
			if(answer2 != ""){
				if _, err := s.Db.Exec("INSERT INTO answer (i_answer, i_question, correct, text) VALUES ((SELECT ifnull(max(i_answer), 0)+1 FROM answer),'" + string(IdQuest) +"','"+type2+"','"+answer2+"')"); err != nil {
					log.Print("Ошибка добавление в базу")
				}
			}
			if(answer3 != ""){
				if _, err := s.Db.Exec("INSERT INTO answer (i_answer, i_question, correct, text) VALUES ((SELECT ifnull(max(i_answer), 0)+1 FROM answer),'" + string(IdQuest) +"','"+type3+"','"+answer3+"')"); err != nil {
					log.Print("Ошибка добавление в базу")
				}
			}
			if(answer4 != ""){
				if _, err := s.Db.Exec("INSERT INTO answer (i_answer, i_question, correct, text) VALUES ((SELECT ifnull(max(i_answer), 0)+1 FROM answer),'" + string(IdQuest) +"','"+type4+"','"+answer4+"')"); err != nil {
					log.Print("Ошибка добавление в базу")
				}
			}


*/







		}
		if(string(createQuestionType) == "2"){
			log.Print("Создание radio вопроса")
			log.Print(createQuestion)
		}

	}
}
func (s *server) CreateTest(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		createTest := []byte(r.FormValue("createTest"))
		createTeststring := string(createTest[:])
		var IdTest = ""
		if _, err := s.Db.Exec("INSERT INTO test_name (i_test, name) VALUES ((SELECT ifnull(max(i_test), 0)+1 FROM test_name),'" + string(r.FormValue("name")) + "')"); err != nil {
			log.Print("Ошибка добавление в базу")
		}
		if err := s.Db.Get(&IdTest, "SELECT MAX(i_test) FROM test_name "); err != nil {
			log.Fatal(err)
		}
		for _, item := range createTeststring {
			if string(item) !=","{
				if _, err := s.Db.Exec("INSERT INTO test_question (i, i_test, i_question) VALUES ((SELECT ifnull(max(i), 0)+1 FROM test_question),'" + string(IdTest) +"','" + string(item) +"')"); err != nil {
					log.Print("Ошибка добавление в базу")
				}
			}
		}
	}
	if r.Method == "GET" {
		session, _ := store.Get(r, config.CookieName)
		// проверка на вторизацию
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			//http.Error(w, "Не авторизованный", http.StatusForbidden)
			fmt.Println("Не авторизованный")
			//http.Redirect(w, r, "/login/", 302)
			return
		}
		if (session.Values["admin"] != "admin") {
			http.Redirect(w, r, "/", 302)
			return
		}

		var TestQuestion []TestQuestionAdmin

		err := s.Db.Select(&TestQuestion, `SELECT i_question, question_name FROM question ORDER BY i_question ASC `)
		if err != nil {
			http.Error(w, http.StatusText(500), 500)
			fmt.Println(err)
			return
		}
		log.Println(TestQuestion)
		fmt.Fprint(w, templates.CreateTestPage(TestQuestion))
		//fmt.Fprintln(w, "Авторизованный")
	}
}

func (s *server) ResultAdminTest(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, config.CookieName)
	// проверка на вторизацию
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		//http.Error(w, "Не авторизованный", http.StatusForbidden)
		http.Redirect(w, r, "/login/", 302)
		return
	}
	if (session.Values["admin"] != "admin") {
		http.Redirect(w, r, "/", 302)
		return
	}

	var test TestAnswerAdmin
	if err := s.Db.Get(&test, "SELECT * FROM user_answer WHERE id  = $1", r.FormValue("id")); err != nil {
		log.Print(err)
		return
	}
	log.Println(test)
	log.Println(config.PathTestResult + "user_" + test.IdUser + "/id_test_" + test.IdTest + "/answer_" + test.Id + "_with.json")
	log.Println(config.PathTestResult + "user_" + test.IdUser + "/id_test_" + test.IdTest + "/answer_" + test.Id + "_without.json")


	jsonFile, err := os.Open(config.PathTestResult + "user_" + test.IdUser + "/id_test_" + test.IdTest + "/answer_" + test.Id + "_without.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened users.json")
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var AnswerUserJson []TestQuestionAnswer

	json.Unmarshal(byteValue, &AnswerUserJson)
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	log.Println(AnswerUserJson)

	fmt.Fprint(w, templates.ResultTestAdmin(AnswerUserJson))
	//fmt.Fprintln(w, "Авторизованный")
}

/*func (s *server) TestIndex(w http.ResponseWriter, r *http.Request) { //все тесты
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
}*/
func main() {
	if err := loadConfig(); err != nil {
		log.Fatal(err)
	}
	log.Println("Config loaded from", *configFile)
	var id int

	var dbconnect = "postgresql://" + config.DbLogin + ":" + config.DbPassword + "@" + config.DbHost + ":26257/" + config.DbName + "?sslmode=disable"
	s := server{Db: sqlx.MustConnect("postgres", dbconnect)}
	defer s.Db.Close()

	if err := s.Db.Get(&id, "SELECT count(*) FROM users"); err != nil {
		log.Fatal(err)
	}

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/users/", s.UsersIndex)
	http.HandleFunc("/users/show/", s.UsersShow)
	http.HandleFunc("/testStart/", s.TestStart)
	http.HandleFunc("/testCheckQuestion/", s.TestCheckQuestion)
	http.HandleFunc("/testCheckQuestionWhith/", s.TestCheckQuestionWith)

	http.HandleFunc("/logout/", s.logout)
	http.HandleFunc("/login/", s.login)

	http.HandleFunc("/startVds/", s.CreatedVds)
	http.HandleFunc("/stopVds/", s.StopdVds)
	http.HandleFunc("/", s.Index)

	http.HandleFunc("/admin", s.IndexAdmin)
	http.HandleFunc("/create-test/", s.CreateTest)
	http.HandleFunc("/create-question", s.CreateQuestion)






	http.HandleFunc("/admin/result/test", s.ResultAdminTest)

	//http.HandleFunc("/getTest/", s.TestIndex)
	http.ListenAndServe(":4080", nil)
}
