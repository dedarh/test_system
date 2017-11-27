package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"flag"
	"io/ioutil"
)


var config struct {
	DbLogin      string `json:"dbLogin"`
	DbHost       string `json:"dbHost"`
	DbDb         string `json:"dbDb"`
}


type User struct {
	Id       int    `db:"id_user"`
	Login    string `db:"login"`
	Password string `db:"password"`
	Name     string `db:"name"`
	Gr       int    `db:"gr"`
	Token    string `db:"token"`
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
}

var db *sqlx.DB
var configFile  = flag.String("config", "conf.json", "Where to read the config from")


func loadConfig() error {
	jsonData, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, &config)
}

func init() {
	if err := loadConfig(); err != nil {
		log.Fatal(err)
	}
	log.Println("Config loaded from", *configFile)
		var id int
		var dbconnect =	"postgresql://"+config.DbLogin+"@"+config.DbHost+":26257/"+config.DbDb+"?sslmode=disable"

		db =	sqlx.MustConnect("postgres", dbconnect)

	// 	db =	sqlx.MustConnect("postgres", "postgresql://root@localhost:26257/test_systems?sslmode=disable") //govnocode

	if err := db.Get(&id, "SELECT count(*) FROM users"); err != nil {
		log.Fatal(err)
	}
	log.Print("количество пользователей", id)
	//fmt.Printf("%#v\n%#v","количество пользователей", id)
}

func usersIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	var users []User

	if err := db.Select(&users, "SELECT * FROM users ORDER BY Name ASC"); err != nil {
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

func usersShow(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	var user []User

	if err := db.Select(&user, "SELECT * FROM users WHERE id_user  = $1", r.FormValue("id")); err != nil {
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

func testIndex(w http.ResponseWriter, r *http.Request) { //все тесты
	if r.Method != "POST" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	var test []Test


	if err := db.Select(&test, "SELECT * FROM test_name  ORDER BY name ASC"); err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	answer_test, err:= json.Marshal(test)  //govnocode
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintf(w, string(answer_test))
}

func testStart(w http.ResponseWriter, r *http.Request) { //все тесты шаблон
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	id := r.FormValue("id") //id теста

	var testQuestion []Test_question
	err := db.Select(&testQuestion, `SELECT q.question_name,t_q.i_question, q.type FROM "test_question" t_q JOIN "question" q ON t_q.i_question = q.i_question  WHERE t_q.i_test = $1 ORDER BY q.i_question DESC`, id)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		fmt.Println(err)
		return
	}

	for i := 1; i < len(testQuestion); i++ {
		var tqa []Test_question_answer
		db.Select(&tqa, "SELECT i_question, i_answer, text FROM answer WHERE i_question = $1",testQuestion[i].Id_Question)
		testQuestion[i].Answer = tqa
	}







	fmt.Println(testQuestion)
	answer_test, err := json.Marshal(testQuestion)  //govnocode
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintf(w, string(answer_test))
}

func nextQuestion(w http.ResponseWriter, r *http.Request) { //все тесты шаблон
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	var question_answer []Test_question_answer


	if err := db.Select(&question_answer, `SELECT i_question, i_answer, text FROM answer WHERE i_question = $1`, r.FormValue("Id_Question")); err != nil {
		http.Error(w, http.StatusText(500), 500)
		fmt.Println(err)
		return
	}

	fmt.Println(question_answer)
	answer_test, err := json.Marshal(question_answer)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintf(w, string(answer_test))
}

func main() {
	http.HandleFunc("/users/", usersIndex)
	http.HandleFunc("/users/show/", usersShow)
	http.HandleFunc("/get_test/", testIndex)
	http.HandleFunc("/testStart/", testStart)
	http.HandleFunc("/next/", nextQuestion)
	http.ListenAndServe(":4080", nil)
}
