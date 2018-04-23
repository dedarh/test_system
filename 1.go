package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"flag"
	"io/ioutil"
	"./templates"
	"os"
	"text/template"
	"os/exec"
	"strconv"
)

type server struct {
	Db *sqlx.DB
}

type Path struct {
	FirstPart  string
	SecondPart string
}

var config struct {
	DbLogin        string `json:"dbLogin"`
	DbHost         string `json:"dbHost"`
	DbDb           string `json:"dbDb"`
	PathToConfVm   string `json:"PathToConfVm"`
	PathTestResult string `json:"PathTestResult"`
}

type answer_return struct {
	CountCorrect int
	CountWrong   int
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
type Test_question struct {
	Id_Question string `db:"i_question"`
	Text        string `db:"question_name"`
	Type        string `db:"type"`
	Answer      []Test_question_answer
}
type Test_question_answer struct {
	Id_Question int    `db:"i_question"`
	ID_Answer   int    `db:"i_answer"`
	Text        string `db:"text"`
	Correct     int    `db:"status"`
}

var db *sqlx.DB
var configFile = flag.String("config", "conf.json", "Where to read the config from")

func loadConfig() error {
	jsonData, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, &config)
}

func (s *server) getUserFromDbByLogin(login string) (User, error) {
	var users User
	var query = "SELECT * FROM users WHERE users.login='" + login + "'";
	err := s.Db.Get(&users, query)
	return users, err
}

func (s *server) makeVagrantConf(login string) error {
	usr, err := s.getUserFromDbByLogin(login)
	if err != nil {
		return err
	}
	log.Print("юзер")
	log.Print(usr)
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
	log.Println(config.PathToConfVm + string(usr.Login) + "/" + usr.Login + "/Vagrantfile")
	defer file.Close()
	templates.ExecuteTemplate(file, "VagrantConfSample.txt", generatedVm)

	s.makeBat(login)
	return err
}

func (s *server) makeBat(login string) error {
	usr, err := s.getUserFromDbByLogin(login)
	text := "vagrant up"
	file, err := os.Create(config.PathToConfVm + string(usr.Login) + "/" + usr.Login + "/Vagrantfile.bat")
	file.WriteString(text)
	defer file.Close()
	return err
}

func (s *server) executeVagrant(path string) error {
	log.Println(path)
	vagrant := exec.Command(path + "Vagrantfile.bat")
	vagrant.Dir = path
	err := vagrant.Run()
	return err
}

func (s *server) CreatedVds(w http.ResponseWriter, r *http.Request) {
	var login = r.FormValue("user")
	log.Println()
	err := s.makeVagrantConf(login)
	s.executeVagrant(config.PathToConfVm + login + "/" + login + "/")
	if err != nil {
		log.Println(err)
		return
	}
	var query = "INSERT INTO suggestions (id, login, state, status) VALUES ((SELECT ifnull(max(id), 0)+1 FROM suggestions),'" + login + "', 0,0)";
	if _, err := s.Db.Exec(query); err != nil {
		log.Print("Ошибка добавление в базу")
		http.Error(w, http.StatusText(500), 500)
		return
	}
}

func (s *server) usersIndex(w http.ResponseWriter, r *http.Request) { //игформация о пользователе
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	var users []User

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
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	var user []User

	if err := s.Db.Select(&user, "SELECT * FROM users WHERE id_user  = $1", r.FormValue("id")); err != nil {
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
	if r.Method != "POST" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	var test []Test

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
		id := r.FormValue("id") //id теста

		var testQuestion []Test_question
		err := s.Db.Select(&testQuestion, `SELECT q.question_name,t_q.i_question, q.type FROM "test_question" t_q JOIN "question" q ON t_q.i_question = q.i_question  WHERE t_q.i_test = $1 ORDER BY q.i_question DESC`, id)
		if err != nil {
			http.Error(w, http.StatusText(500), 500)
			fmt.Println(err)
			return
		}

		for i := 1; i < len(testQuestion); i++ {
			var tqa []Test_question_answer
			s.Db.Select(&tqa, "SELECT i_question, i_answer, text FROM answer WHERE i_question = $1", testQuestion[i].Id_Question)
			testQuestion[i].Answer = tqa
		}
		fmt.Println(testQuestion)
		answer_test, err := json.Marshal(testQuestion) //govnocode
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Fprintf(w, string(answer_test))
	}

}

func (s *server) test_check_qestion_whith(w http.ResponseWriter, r *http.Request) { //проверка верного ответа на вопрос  http://localhost:4080/test_check_qestion/?Id_Test=1&Answer_user=[{"Id_Question":1,"ID_Answer":1,"Text":"test%20text"},{"Id_Question":1,"ID_Answer":2,"Text":"test%20text"},{"Id_Question":1,"ID_Answer":3,"Text":"test%20text"},{"Id_Question":2,"ID_Answer":5,"Text":"%20"}]
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}
	session, _ := store.Get(r, "cookie-name")

	var Answer_user_json []Test_question_answer
	user_answer := []byte(r.FormValue("Answer_user"))

	err := json.Unmarshal(user_answer, &Answer_user_json)
	if err != nil {
		log.Println(err)
		return
	}
	var id_user = strconv.Itoa(session.Values["id"].(int))
	var id_file = ""
	if err := s.Db.Get(&id_file, "SELECT ifnull(max(id)+1, 0) FROM user_answer"); err != nil {
		log.Fatal(err)
	}

	os.Mkdir(config.PathTestResult+"user_"+id_user+"/", 0777)
	os.Mkdir(config.PathTestResult+"user_"+id_user+"/id_test_"+r.FormValue("Id_Test")+"/", 0777)
	file, err := os.Create(config.PathTestResult + "user_" + id_user + "/id_test_" + r.FormValue("Id_Test") + "/answer_"+id_file+"_with.json")
	file.Write(user_answer)
	file.Close()
}

func (s *server) test_check_qestion(w http.ResponseWriter, r *http.Request) { //проверка верного ответа на вопрос  http://localhost:4080/test_check_qestion/?Id_Test=1&Answer_user=[{"Id_Question":1,"ID_Answer":1,"Text":"test%20text"},{"Id_Question":1,"ID_Answer":2,"Text":"test%20text"},{"Id_Question":1,"ID_Answer":3,"Text":"test%20text"},{"Id_Question":2,"ID_Answer":5,"Text":"%20"}]
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}
	session, _ := store.Get(r, "cookie-name")
	var countCorrect = 0;
	var countWrong = 0;
	var question_answer_correct []Test_question_answer

	var Answer_user_json []Test_question_answer
	user_answer := []byte(r.FormValue("Answer_user"))

	err := json.Unmarshal(user_answer, &Answer_user_json)
	if err != nil {
		log.Println(err)
		return
	}
	for i := 0; i < len(Answer_user_json); i++ {
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
	}
	log.Println("Верных ответов: ", countCorrect)
	log.Println("Неверных ответов: ", countWrong)
	var return_obj = map[string]int{"countCorrect": countCorrect, "countWrong": countWrong}
	mapVar2, _ := json.Marshal(return_obj)

	var id_user = strconv.Itoa(session.Values["id"].(int))
	var query = "INSERT INTO user_answer (id, id_user, id_test, answer_temp) VALUES ((SELECT ifnull(max(id), 0)+1 FROM user_answer), '" + id_user + "' , '" + r.FormValue("Id_Test") + "','" + string(mapVar2) + "')";
	if _, err := s.Db.Exec(query); err != nil {
		log.Print("Ошибка добавление в базу")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	var id_file = ""
	if err := s.Db.Get(&id_file, "SELECT ifnull(max(id), 0) FROM user_answer"); err != nil {
		log.Fatal(err)
	}

	os.Mkdir(config.PathTestResult+"user_"+id_user+"/", 0777)
	os.Mkdir(config.PathTestResult+"user_"+id_user+"/id_test_"+r.FormValue("Id_Test")+"/", 0777)
	file, err := os.Create(config.PathTestResult + "user_" + id_user + "/id_test_" + r.FormValue("Id_Test") + "/answer_"+id_file+"_without.json")

	file.Write(user_answer)
	file.Close()
	fmt.Fprintf(w, string(mapVar2))
}

var (
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	key   = []byte("super-secret-key")
	store = sessions.NewCookieStore(key)
)

func (s *server) secret_test(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "cookie-name")
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
	session, _ := store.Get(r, "cookie-name")
	var user []User

	if r.Method == "GET" {
		fmt.Fprint(w, templates.LoginPage())
	}

	if r.Method == "POST" {
		log.Print(r)
		log.Print(r.FormValue("login"))
		var query = "SELECT * FROM users WHERE users.password  = '" + r.FormValue("password") + "' and users.login = '" + r.FormValue("login") + "'";
		log.Print(query)

		if err := s.Db.Select(&user, query); err != nil {
			http.NotFound(w, r)
			return
		}

		session.Values["authenticated"] = true
		session.Values["id"] = user[0].Id
		session.Values["names"] = user[0].Firstname
		session.Values["lastname"] = user[0].Lastname
		session.Values["admin"] = user[0].Permission
		session.Values["group"] = user[0].Gr
		session.Save(r, w)
		http.Redirect(w, r, "/secret_test/", 302)
	}

}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "cookie-name")

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
	http.HandleFunc("/test_check_qestion/", s.test_check_qestion)
	http.HandleFunc("/test_check_qestion_whith/", s.test_check_qestion_whith)

	http.HandleFunc("/logout/", s.logout)
	http.HandleFunc("/login/", s.login)

	http.HandleFunc("/start_vds/", s.CreatedVds)
	http.HandleFunc("/", s.secret_test)
	http.ListenAndServe(":4080", nil)
}
