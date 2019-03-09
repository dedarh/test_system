var App = {
    noty: function(status, text) {
        $('.alert-' + status + ' em').text(text);
        $('.alert-' + status).show();
        setTimeout(function() {
            $('.alert-' + status).hide();
        }, 4000);
    },
    var: {
        id_test: 0,
        load_questions: {},
        questions: [],
        user_ansver: []
    },
    user: {
        question_id: 0,
    },
    systems: {
		create_test: function() {
			var val = $("#inputLogin").val();
            var temp1 = $(".panel.panel-info :checked");
            var user_ansver = [];           
               for (var i = 0; i < temp1.length; i++) {
                    user_ansver.push(Number(temp1[i].attributes.id_q.value));
                }
            
            $.ajax({
                url: '/create-test/?createTest=' + user_ansver + '&name=' + val +'',
                type: 'post',
                success: function(data) {
                    App.noty('success', 'Cоздание теста завершена');
                    
                },
                error: function() {
                    App.noty('danger', 'Ошибка при отправке данных на API!');
                }
            });				
        },
        get_test: function() {
            $.ajax({
                url: '/getTest/',
                type: 'post',
                success: function(data) {
                    data = JSON.parse(data);
                    if (data.error) {
                        App.noty('danger', 'нет тестов!');
                    } else {
                        var html = "";
                        for (var i = 0; i < data.length; i++) {
                            html += " <tr class=\"info\"><td style=\"padding: 20px;\"> " + data[i].Id + "</td><td style=\"padding: 20px;\">" + data[i].Name + "</td><td><a href=\"/testStart/?id=" + data[i].Id + "\" class=\"btn btn-success btn-on-table\"><span class=\"glyphicon glyphicon-play\" aria-hidden=\"true\"></span></a></td></tr>";
                        }
                        $('tbody').html(html);
                        App.noty('success', 'Закгрузка тестов завершена');
                    }
                },
                error: function() {
                    App.noty('danger', 'Ошибка при отправке данных на API!');
                }
            });
        },
        load_question: function() {
            $.ajax({
                url: '/testStart/?id=' + App.var.id_test + '',
                type: 'post',
                success: function(data) {
                    //console.log(data);
                    App.var.load_questions = JSON.parse(data);
                    App.systems.print_test(App.var.load_questions);
                },
                error: function() {
                    console.log('danger Ошибка при отправке данных на API!');
                }
            });
        },
        print_test: function(question) {
            var html = '';
            for (var i = 0; i < question.length; i++) {
                if (question[i].Answer == null) {
                    html += '<div class="panel panel-info"><div class="panel-heading"><h3 class="panel-title">';
                    html += question[i].Text;
                    html += '</h3></div> <div class="panel-body"><form class="form-horizontal form-cont"><fieldset><div class="form-group is-empty"><label for="inputLogin" class="col-md-2 control-label">Ansver</label> <div class="col-md-10"><input id_q="' + question[i].IdQuestion + ' "type="login" name="login" class="form-control" id="inputLogin"></div></div></fieldset></form></div></div>';
                }
                if (question[i].Type == "2" && question[i].Answer != null) {
                    html += '<div id_q="' + question[i].IdQuestion + '"class="panel panel-info"> <div class="panel-heading"><h3 class="panel-title">';
                    html += question[i].Text;
                    html += '</h3> </div>';
                    html += '<div class="panel-body">';
                    html += '<form class="form-horizontal form-cont">';
                    html += '<fieldset>';
                    for (var j = 0; j < question[i].Answer.length; j++) {
                        html += '<div class="radio"><label><input id_q="' + question[i].Answer[j].IdQuestion + '" id_answer="' + question[i].Answer[j].IdAnswer + '"type="radio" name="optradio"><span class="circle"></span><span class="check"></span>' + question[i].Answer[j].Text + '</label></div>';
                    }
                    html += '</fieldset></form></div></div>';
                }
                if (question[i].Type == "1" && question[i].Answer != null) {
                    html += '<div id_q="' + question[i].IdQuestion + '" class="panel panel-info"><div class="panel-heading"><h3 class="panel-title">';
                    html += question[i].Text;
                    html += '</h3></div> <div class="panel-body"><form class="form-horizontal form-cont"><fieldset> <div class="form-group"><div class="col-md-10">';
                    for (var j = 0; j < question[i].Answer.length; j++) {
                        html += '<div class="checkbox"><label><input id_q="' + question[i].Answer[j].IdQuestion + '" id_answer="' + question[i].Answer[j].IdAnswer + '" type="checkbox"><span class="checkbox-material"><span class="check"></span></span>' + question[i].Answer[j].Text + '</label></div>    ';
                        }
                    html += '</div></div></fieldset></form></div></div>';
                }

            }
            html += '';
            $("#questblock").html("");
            $("#questblock").append($(html));
            App.user.question_id = 1;
        },
        next_question: function() {
            var user_ansver = [];
            var user_ansver_with = [];
            var temp1 = $(".panel.panel-info :checked");
            for (var i = 0; i < temp1.length; i++) {
                var temp2 = {};
                temp2.IdQuestion = Number(temp1[i].attributes.id_q.value);
                temp2.IdAnswer = Number(temp1[i].attributes.id_answer.value);
                temp2.Text = temp1[i].nextSibling.nodeValue;
                user_ansver.push(temp2);
            }
            var temp3 = $("input[id='inputLogin']");
            for (var i = 0; i < temp3.length; i++) {
                var temp = {};
                temp.IdQuestion = Number(temp3[i].attributes.id_q.value);
                temp.Text = temp3[i].value;
                user_ansver_with.push(temp);
            }
            $.ajax({
                url: '/testCheckQuestion/?Id_Test=' + App.var.id_test + '&Answer_user=' + JSON.stringify(user_ansver) + '',
                type: 'post',
                success: function(data) {
                    data = JSON.parse(data);
                    $('#questblock').hide();
                    $('#Finish').hide();
                    App.noty('success', "Верных ответов " + data.countCorrect + "");
                    App.noty('danger', "Не верных ответов " + data.countWrong + "");
                    setTimeout("window.location.replace(\"http://localhost:4080\");", 4000);
                },
                error: function() {
                    console.log('danger Ошибка при отправке данных на API!');
                }
            });
            $.ajax({
                url: '/testCheckQuestionWhith/?Id_Test=' + App.var.id_test + '&Answer_user=' + JSON.stringify(user_ansver_with) + '',
                type: 'post',
                error: function() {
                    console.log('danger Ошибка при отправке данных на API!');
                }
            });
        },
    }
};
