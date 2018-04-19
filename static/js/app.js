var App = {
    noty: function(status, text) {
        $('.alert-' + status + ' em').text(text);
        $('.alert-' + status).show();
        setTimeout(function() {
            $('.alert-' + status).hide();
        }, 4000);
    },
    var: {

    },
    user: {

    },
    systems: {
        get_test: function() {
            $.ajax({
                url: '/get_test/',
                type: 'post',
                success: function(data) {
                    data = JSON.parse(data);
                    if(data.error){
                        App.noty('danger', 'нет тестов!');
                    }else{
                        var html = "";
                        for(var i=0; i<data.length; i++){
                            html+= " <tr class=\"info\"><td style=\"padding: 20px;\"> "+data[i].Id+"</td><td style=\"padding: 20px;\">"+data[i].Name+"</td><td><a href=\"/testStart/?id="+data[i].Id+"\" class=\"btn btn-success btn-on-table\"><span class=\"glyphicon glyphicon-play\" aria-hidden=\"true\"></span></a></td></tr>";
                        }
                        $('tbody').html(html);

                        console.log(data);
                        App.noty('success', 'Закгрузка тестов завершена');
                    }
                },
                error: function() {
                    App.noty('danger', 'Ошибка при отправке данных на API!');
                }
            });
        }
    }
};
$(function() {
    $.fn.extend({
        animateCss: function(animationName) {
            var animationEnd = 'webkitAnimationEnd mozAnimationEnd MSAnimationEnd oanimationend animationend';
            $(this).addClass('animated ' + animationName).one(animationEnd, function() {
                $(this).removeClass('animated ' + animationName);
            });
        },
        slowHide: function(animationName) {
            var animationEnd = 'webkitAnimationEnd mozAnimationEnd MSAnimationEnd oanimationend animationend';
            $(this).addClass('animated ' + animationName).one(animationEnd, function() {
                $(this).hide();
                $(this).removeClass('animated ' + animationName);
            });
        },
        slowShow: function(animationName) {
            var animationEnd = 'webkitAnimationEnd mozAnimationEnd MSAnimationEnd oanimationend animationend';
            $(this).show();
            $(this).addClass('animated ' + animationName).one(animationEnd, function() {
                $(this).removeClass('animated ' + animationName);
            });
        }
    });

});