{{define "js-checkboxes"}}

function toggleNamedCheckboxes(name) {
    checkboxes = document.getElementsByName(name);
    for(var i=0, n=checkboxes.length;i<n;i++) {
        checkboxes[i].checked = !checkboxes[i].checked; //source.checked;
    }
}

function assignNamedCheckboxes(name, assignedValue) {
    checkboxes = document.getElementsByName(name);
    for(var i=0, n=checkboxes.length;i<n;i++) {
        checkboxes[i].checked = assignedValue
    }
}

function clearNamedCheckboxes(name) { assignNamedCheckboxes(name,false) }
function setNamedCheckboxes(name)   { assignNamedCheckboxes(name,true) }

{{end}}
