// This file is automatically generated by qtc from "test.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line templates\test.qtpl:1
package templates

//line templates\test.qtpl:1
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line templates\test.qtpl:1
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line templates\test.qtpl:1
func StreamTestsPageView(qw422016 *qt422016.Writer, idtest string) {
	//line templates\test.qtpl:1
	qw422016.N().S(`
`)
	//line templates\test.qtpl:2
	StreamHead(qw422016, "Tест начат")
	//line templates\test.qtpl:2
	qw422016.N().S(`
<div class="navbar navbar-inverse">
        <div class="container-fluid">
            <div class="navbar-header">
                <button type="button" class="navbar-toggle" data-toggle="collapse"
                        data-target=".navbar-responsive-collapse">
                    <span class="icon-bar"></span>
                    <span class="icon-bar"></span>
                    <span class="icon-bar"></span>
                </button>
                <a class="navbar-brand" href="/">Testing system</a>
            </div>
            <div class="collapse navbar-collapse" id="bs-example-navbar-collapse-1">
                <ul class="nav navbar-nav">
                    <li><a href="/tests/">Tests</a></li>
                </ul>
                <ul class="nav navbar-nav navbar-right">
                    <li><a href="/admin/logout/">Logout</a></li>
                </ul>
            </div>
        </div>
    </div>
		
		
	<div id ="questblock" style="\width: 80%; margin: 0 auto;min-width: 1320px\"> </div>
<button id="Finish" class="btn btn-primary left">Завершить тест<div class="ripple-container"></div></button>
		
		
		
<script>
    $(document).ready(function(){
        App.var.id_test= `)
	//line templates\test.qtpl:33
	qw422016.E().S(idtest)
	//line templates\test.qtpl:33
	qw422016.N().S(`;
	    App.systems.load_question();		
	    $("body").append($("<div id = \"questblock\"></div>"));		
		$('#Finish').click(function(e) {        
			App.systems.next_question();
		});

		

		
      });
</script>
`)
	//line templates\test.qtpl:45
	StreamFooter(qw422016)
	//line templates\test.qtpl:45
	qw422016.N().S(`
`)
//line templates\test.qtpl:46
}

//line templates\test.qtpl:46
func WriteTestsPageView(qq422016 qtio422016.Writer, idtest string) {
	//line templates\test.qtpl:46
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line templates\test.qtpl:46
	StreamTestsPageView(qw422016, idtest)
	//line templates\test.qtpl:46
	qt422016.ReleaseWriter(qw422016)
//line templates\test.qtpl:46
}

//line templates\test.qtpl:46
func TestsPageView(idtest string) string {
	//line templates\test.qtpl:46
	qb422016 := qt422016.AcquireByteBuffer()
	//line templates\test.qtpl:46
	WriteTestsPageView(qb422016, idtest)
	//line templates\test.qtpl:46
	qs422016 := string(qb422016.B)
	//line templates\test.qtpl:46
	qt422016.ReleaseByteBuffer(qb422016)
	//line templates\test.qtpl:46
	return qs422016
//line templates\test.qtpl:46
}