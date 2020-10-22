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

//ToggleLibraryMenu toggles library nodes
function ToggleLibraryMenu(pageID, naviPlus) {
    //First, determine toggle state
    if (naviPlus.innerHTML.includes("-")) {
        //Toggle state is open, so we need to close
        $(naviPlus).html("+")
        //Find parent li
        emptyUL = document.createElement("ul")
        $($(naviPlus).parents("li")[0]).children("ul").replaceWith(emptyUL)
        return false;
    }

    fetch("/api/notes/"+pageID+"/children")
        .then((resp) => resp.json()) //Convert response to json
        .then(function(jsonData) {
            SetLibraryMenuNode(jsonData.Data, naviPlus)
        })
        .catch(function(err) {
            console.log(err);
        });
    return false; //Prevent hyperlinks from activating
}

//Should be called by ToggleLibraryMenu. Wipes the existing ul, and replaces with updated menu
function SetLibraryMenuNode(libraryData, naviPlus) {
    //Create the new ul
    NewUL = document.createElement("ul")
    //Add the LIs to the new ul
    if (libraryData.Children) {
        for (i = 0; i<libraryData.Children.length; i++) {
            //Create elements
            NewA = document.createElement("a")
            NewA.href="/page/"+libraryData.Children[i].ID+"/view"

            NewLI = document.createElement("li")

            NewPlus = document.createElement("div")
            NewPlus.classList.add("naviPlus")
            NewPlus.appendChild(document.createTextNode("+"))
            $(NewPlus).on("click", {ID: libraryData.Children[i].ID, newPlus: NewPlus}, function(eventData) {return ToggleLibraryMenu(eventData.data.ID, eventData.data.newPlus);} );

            NewSpan = document.createElement("span")
            NewSpan.classList.add("naviLabel")
            NewSpan.appendChild(document.createTextNode(libraryData.Children[i].Name))

            NewLI.appendChild(NewA)
            NewA.appendChild(NewPlus)
            NewA.appendChild(NewSpan)
            NewLI.appendChild(document.createElement("ul"))
            NewUL.appendChild(NewLI);
        }
    }

    if (LoggedIn) {
        //Add create page node
        newCreatePage = document.getElementById("createPageTemplate").content.firstElementChild.cloneNode(true)
        $(newCreatePage).find("input[name='ParentID']").get(0).value = libraryData.CurrentPage.ID
        NewUL.appendChild(newCreatePage);
    }
    //Toggle state is closed, so we need to open
    $(naviPlus).html("-")
    //Find parent li
    $($(naviPlus).parents("li")[0]).children("ul").replaceWith(NewUL)
}

//AddCreateNodes adds Create Note nodes to the menu on first load
function AddCreateNodes() {
    if (LoggedIn) {
        toReplace = $(".confirmCSRF")
        for (i = 0; i<toReplace.length; i++) {
            parentID = $(toReplace[i]).find("input[name='ParentID']").get(0).value

            newCreatePage = document.getElementById("createPageTemplate").content.firstElementChild.cloneNode(true)
            $(newCreatePage).find("input[name='ParentID']").get(0).value = parentID
            $(toReplace[i]).replaceWith(newCreatePage)
        }
    }
}

$(document).ready(AddCreateNodes)