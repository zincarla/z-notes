//ToggleDIVDisplay is used to toggle a display from hidden to visible or vice versa. Used in showing user a message from the template input
function ToggleDIVDisplay(id) {
    if (document.getElementById(id).classList.contains("displayBlock") || document.getElementById(id).classList.contains("displayHidden") != true) {
        document.getElementById(id).classList.add("displayHidden");
        document.getElementById(id).classList.remove("displayBlock");
    } else {
        document.getElementById(id).classList.add("displayBlock");
        document.getElementById(id).classList.remove("displayHidden");
    }
}

//toggleCSSClass is used to toggle a css class on a given html element.
function toggleCSSClass(elementID, className) {
    if (document.getElementById(elementID).classList.contains(className)) {
        document.getElementById(elementID).classList.remove(className);
    } else {
        document.getElementById(elementID).classList.add(className);
    }
}