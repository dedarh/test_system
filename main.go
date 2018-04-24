package main

import (
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
	DbLogin        string `json:"dbLogin"`
	DbHost         string `json:"dbHost"`
	DbDb           string `json:"dbDb"`
	PathToConfVm   string `json:"PathToConfVm"`
	PathTestResult string `json:"PathTestResult"`
	CookieName     string `json:"CookieName"`
	Key            string `json:"Key"`
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
	Id_Question string `db:"i_question"`
	Text        string `db:"question_name"`
	Type        string `db:"type"`
	Answer      []TestQuestionAnswer
}
type TestQuestionAnswer struct {
	Id_Question int    `db:"i_question"`
	ID_Answer   int    `db:"i_answer"`
	Text        string `db:"text"`
	Correct     int    `db:"status"`
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
func (s *server) makeVagrantConf(login string) error {
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

	errFile := s.makeBat(login)
	if errFile != nil {
		log.Println(err)
	}
	return err
}
func (s *server) makeBat(login string) error {
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
func (s *server) executeVagrant(path string) error {
	log.Println(path)
	vagrant := exec.Command(path + "Vagrantfile.bat")
	vagrant.Dir = path
	err := vagrant.Run()
	return err
}
func (s *server) executeVagrantHalt(path string) error {
	vagrant := exec.Command(path + "Vagranthalt.bat")
	vagrant.Dir = path
	err := vagrant.Run()
	return err
}
func (s *server) createdVds(w http.ResponseWriter, r *http.Request) {
	var login = r.FormValue("user")
	err := s.makeVagrantConf(login)
	s.executeVagrant(config.PathToConfVm + login + "/" + login + "/")
	if err != nil {
		log.Println(err)
		return
	}
	if _, err := s.Db.Exec("INSERT INTO suggestions (id, login, state, status) VALUES ((SELECT ifnull(max(id), 0)+1 FROM suggestions),'" + login + "', 0,0)"); err != nil {
		log.Print("Ошибка добавление в базу")
		return
	}
}
func (s *server) stopdVds(w http.ResponseWriter, r *http.Request) {
	var login = r.FormValue("user")
	error := s.executeVagrantHalt(config.PathToConfVm + login + "/" + login + "/")
	if (error != nil) {
		log.Println(error)
		return
	}
}
func (s *server) usersIndex(w http.ResponseWriter, r *http.Request) { //игформация о пользователе
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
		answer_user, err := json.Marshal(u) //govnocode
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Fprintf(w, string(answer_user))
	}
}
func (s *server) usersShow(w http.ResponseWriter, r *http.Request) { //игформация об определенном пользователе
	var user []User
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	if err := s.Db.Get(&user, "SELECT * FROM users WHERE id_user  = $1", r.FormValue("id")); err != nil {
		http.NotFound(w, r)
		return
	}

	answer_user, err := json.Marshal(user[0]) //govnocode
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintf(w, string(answer_user))
}
func (s *server) testIndex(w http.ResponseWriter, r *http.Request) { //все тесты
	var test []Test
	if r.Method != "POST" {
		http.Error(w, http.StatusText(405), 405)
		return
	}
	if err := s.Db.Select(&test, "SELECT * FROM test_name  ORDER BY name ASC"); err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}
	answer_test, err := json.Marshal(test) //govnocode
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintf(w, string(answer_test))
}
func (s *server) testStart(w http.ResponseWriter, r *http.Request) { //создание теста и отправка его пользователю
	if r.Method == "GET" {
		id := r.FormValue("id")
		fmt.Fprint(w, templates.TestsPageView(id))
	}
	if r.Method == "POST" {
		var testQuestion []TestQuestion
		id := r.FormValue("id") //id теста
		err := s.Db.Select(&testQuestion, `SELECT q.question_name,t_q.i_question, q.type FROM "test_question" t_q JOIN "question" q ON t_q.i_question = q.i_question  WHERE t_q.i_test = $1 ORDER BY q.i_question DESC`, id)
		if err != nil {
			http.Error(w, http.StatusText(500), 500)
			fmt.Println(err)
			return
		}
		for index, e := range testQuestion {
			var tqa []TestQuestionAnswer
			s.Db.Select(&tqa, "SELECT i_question, i_answer, text FROM answer WHERE i_question = $1", e.Id_Question)
			testQuestion[index].Answer = tqa
		}
		answer_test, err := json.Marshal(testQuestion)
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Fprintf(w, string(answer_test))
	}

}
func (s *server) testCheckQestionWhith(w http.ResponseWriter, r *http.Request) { //проверка верного ответа на вопрос  http://localhost:4080/test_check_qestion/?Id_Test=1&Answer_user=[{"Id_Question":1,"ID_Answer":1,"Text":"test%20text"},{"Id_Question":1,"ID_Answer":2,"Text":"test%20text"},{"Id_Question":1,"ID_Answer":3,"Text":"test%20text"},{"Id_Question":2,"ID_Answer":5,"Text":"%20"}]
	session, _ := store.Get(r, config.CookieName)
	var Answer_user_json []TestQuestionAnswer
	var idUser = strconv.Itoa(session.Values["id"].(int))
	var idFile = ""
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}
	user_answer := []byte(r.FormValue("Answer_user"))
	err := json.Unmarshal(user_answer, &Answer_user_json)
	if err != nil {
		log.Println(err)
		return
	}
	if err := s.Db.Get(&idFile, "SELECT ifnull(COUNT(id)+1, 0) FROM user_answer"); err != nil {
		log.Fatal(err)
	}
	os.Mkdir(config.PathTestResult+"user_"+idUser+"/", 0777)
	os.Mkdir(config.PathTestResult+"user_"+idUser+"/id_test_"+r.FormValue("Id_Test")+"/", 0777)
	file, err := os.Create(config.PathTestResult + "user_" + idUser + "/id_test_" + r.FormValue("Id_Test") + "/answer_" + idFile + "_with.json")
	file.Write(user_answer)
	file.Close()
}
func (s *server) testCheckQestion(w http.ResponseWriter, r *http.Request) { //проверка верного ответа на вопрос  http://localhost:4080/test_check_qestion/?Id_Test=1&Answer_user=[{"Id_Question":1,"ID_Answer":1,"Text":"test%20text"},{"Id_Question":1,"ID_Answer":2,"Text":"test%20text"},{"Id_Question":1,"ID_Answer":3,"Text":"test%20text"},{"Id_Question":2,"ID_Answer":5,"Text":"%20"}]
	session, _ := store.Get(r, config.CookieName)
	var countCorrect = 0;
	var countWrong = 0;
	var question_answer_correct []TestQuestionAnswer
	var Answer_user_json []TestQuestionAnswer
	var idFile = ""
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}
	user_answer := []byte(r.FormValue("Answer_user"))
	err := json.Unmarshal(user_answer, &Answer_user_json)
	if err != nil {
		log.Println(err)
		return
	}

	for index, e := range Answer_user_json {
		if err := s.Db.Select(&question_answer_correct, `SELECT i_question, i_answer, text, status FROM answer WHERE i_question = $1 AND i_answer = $2`, e.Id_Question, e.ID_Answer); err != nil {
			http.Error(w, http.StatusText(500), 500)
			fmt.Println(err)
			return
		}
		if (question_answer_correct[index].Correct == 1) {
			//log.Println("верный овтет")
			countCorrect++
		} else {
			//log.Println("не верный овтет")
			countWrong++
		}
	}

	/*	for i := 0; i < len(Answer_user_json); i++ {
			if err := s.Db.Select(&question_answer_correct, `SELECT i_question, i_answer, text, status FROM answer WHERE i_question = $1 AND i_answer = $2`, Answer_user_json[i].Id_Question, Answer_user_json[i].ID_Answer); err != nil {
				http.Error(w, http.StatusText(500), 500)
				fmt.Println(err)
				return
			}
			if (question_answer_correct[i].Correct == 1) {
				//log.Println("верный овтет")
				countCorrect++
			} else {
				//log.Println("не верный овтет")
				countWrong++
			}
		}*/
	log.Println("Верных ответов: ", countCorrect)
	log.Println("Неверных ответов: ", countWrong)
	var returnObj = map[string]int{"countCorrect": countCorrect, "countWrong": countWrong}
	returnObjJson, _ := json.Marshal(returnObj)
	var idUser = strconv.Itoa(session.Values["id"].(int))
	if _, err := s.Db.Exec("INSERT INTO user_answer (id, id_user, id_test, answer_temp) VALUES ((SELECT ifnull(max(id), 0)+1 FROM user_answer), '" + idUser + "' , '" + r.FormValue("Id_Test") + "','" + string(returnObjJson) + "')"); err != nil {
		log.Print("Ошибка добавление в базу")
		http.Error(w, http.StatusText(500), 500)
		return
	}
	if err := s.Db.Get(&idFile, "SELECT ifnull(COUNT(id), 0) FROM user_answer"); err != nil {
		log.Fatal(err)
	}

	os.Mkdir(config.PathTestResult+"user_"+idUser+"/", 0777)
	os.Mkdir(config.PathTestResult+"user_"+idUser+"/id_test_"+r.FormValue("Id_Test")+"/", 0777)
	file, err := os.Create(config.PathTestResult + "user_" + idUser + "/id_test_" + r.FormValue("Id_Test") + "/answer_" + idFile + "_without.json")

	file.Write(user_answer)
	file.Close()
	fmt.Fprintf(w, string(returnObjJson))
}
func (s *server) index(w http.ResponseWriter, r *http.Request) {
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
		log.Print(r)
		log.Print(r.FormValue("login"))
		var user User
		if err := s.Db.Get(&user, "SELECT * FROM users WHERE password  = $1 and login = $2 LIMIT 1", r.FormValue("password"), r.FormValue("login")); err != nil {
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
		log.Fatal(err)
	}
	log.Println("Config loaded from", *configFile)
	var id int
	var dbconnect = "postgresql://" + config.DbLogin + "@" + config.DbHost + ":26257/" + config.DbDb + "?sslmode=disable"

	s := server{Db: sqlx.MustConnect("postgres", dbconnect),}
	defer s.Db.Close()

	if err := s.Db.Get(&id, "SELECT count(*) FROM users"); err != nil {
		log.Fatal(err)
	}
	log.Print("количество пользователей", id)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/users/", s.usersIndex)
	http.HandleFunc("/users/show/", s.usersShow)
	http.HandleFunc("/get_test/", s.testIndex)
	http.HandleFunc("/testStart/", s.testStart)
	http.HandleFunc("/test_check_qestion/", s.testCheckQestion)
	http.HandleFunc("/test_check_qestion_whith/", s.testCheckQestionWhith)

	http.HandleFunc("/logout/", s.logout)
	http.HandleFunc("/login/", s.login)

	http.HandleFunc("/start_vds/", s.createdVds)
	http.HandleFunc("/stop_vds/", s.stopdVds)
	http.HandleFunc("/", s.index)
	http.ListenAndServe(":4080", nil)
}
