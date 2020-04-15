{{define "js-textboxes"}}

// The box names are defined in the HTML setup pages for each view.
//  legend: top-right
//  details: bottom-left

function PaintDetails(htmlFrag)  { ReplaceBox('details', htmlFrag) }
function PaintLegend(htmlFrag)   { ReplaceBox('legend',  htmlFrag) }

function PaintNotes(htmlFrag)   { ReplaceBox('notes',  htmlFrag) }
function AddNote(htmlFrag)   { AppendBox('notes',  htmlFrag) }

function ReplaceBox(name, htmlFrag) { UpdateBox(name, htmlFrag, false) }
function AppendBox(name, htmlFrag) { UpdateBox(name, htmlFrag, true) }

function UpdateBox(name, htmlFrag, append) {
    var box = document.getElementById(name);
    var div = document.createElement('div');
    div.innerHTML = htmlFrag;

    // Delete prev contents
    if (!append) {
        while (box.hasChildNodes()) {
            box.removeChild(box.lastChild);
        }
    }

    box.appendChild(div);
}

{{end}}
