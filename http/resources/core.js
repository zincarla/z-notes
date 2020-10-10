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

//RefreshLibraryMenu will send an API request to pull the children of pageID. When the API response is received, it will replace the current library menu
function RefreshLibraryMenu(pageID) {
    fetch("/api/notes/"+pageID+"/children")
        .then((resp) => resp.json()) //Convert response to json
        .then(function(jsonData) {
            SetLibraryMenu(jsonData.Data)
        })
        .catch(function(err) {
            console.log(err);
        });
    return false; //Prevent hyperlinks from activating
}

//Should be called by RefreshLibraryMenu. Wipes the side menu, and rebuilds.
function SetLibraryMenu(libraryData) {
    //Create new UL //<a href="/page/{{.ID}}/view"><li><div class="naviPlus" onclick="return RefreshLibraryMenu('{{.ID}}');"> + </div>{{.Name}}</li></a>
    NewUL = document.createElement("ul")
    NewUL.id = "naviMenu"
    //Add the LIs
    if (libraryData.Children) {
        for (i = 0; i<libraryData.Children.length; i++) {
            NewA = document.createElement("a")
            NewA.href="/page/"+libraryData.Children[i].ID+"/view"
            NewLI = document.createElement("li")
            NewPlus = document.createElement("div")
            NewPlus.classList.add("naviPlus")
            NewPlus.appendChild(document.createTextNode("+"))
            $(NewPlus).on("click", {ID: libraryData.Children[i].ID}, function(eventData) {return RefreshLibraryMenu(eventData.data.ID);} );
            NewLI.appendChild(NewPlus)
            NewLI.appendChild(document.createTextNode(libraryData.Children[i].Name))
            NewA.appendChild(NewLI);
            NewUL.appendChild(NewA);
        }
    }
    OldUL = document.getElementById("naviMenu")
    //Replace existing UL
    document.getElementById("naviMenu").parentElement.replaceChild(NewUL, OldUL)

    //Now set back button
    //<a href="/page/{{.PageData.PrevID}}/view"><li class="backPageMenuOption specialSideMenuli"><div class="naviPlus" onclick="return RefreshLibraryMenu('{{.PageData.PrevID}}');">-</div>{{.ParentPageData.Name}}</li></a>
    if (CurrentPageID != libraryData.CurrentPage.ID) {
        NewA = document.createElement("a")
        if (libraryData.CurrentPage.ID == "0") {
            NewA.href="/"
        } else {
            NewA.href="/page/"+libraryData.CurrentPage.ID+"/view"
        }
    }
    NewLI = document.createElement("li")
    NewLI.id="backPageMenuOption"
    NewLI.classList.add("specialSideMenuli")
    if (CurrentPageID == libraryData.CurrentPage.ID) {
        NewLI.classList.add("backPageSpacer")
    }
    if (libraryData.CurrentPage.ID != "0") {
        NewPlus = document.createElement("div")
        NewPlus.classList.add("naviPlus")
        NewPlus.appendChild(document.createTextNode("-"))
        $(NewPlus).on("click", {ID: libraryData.CurrentPage.PrevID}, function(eventData) {return RefreshLibraryMenu(eventData.data.ID);} );
        NewLI.appendChild(NewPlus)
    }
    NewLI.appendChild(document.createTextNode(libraryData.CurrentPage.Name))
    if (CurrentPageID != libraryData.CurrentPage.ID) {
        NewA.appendChild(NewLI);
    } else {
        NewA = NewLI
    }
    //Replace existing backbutton
    OldItem = document.getElementById("backPageMenuOption")
    if (OldItem.parentElement.nodeName.toLowerCase() === "a") {
        OldItem.parentElement.parentElement.replaceChild(NewA, OldItem.parentElement)
    } else {
        OldItem.parentElement.replaceChild(NewA, OldItem)
    }
}