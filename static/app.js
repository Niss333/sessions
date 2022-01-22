//
function ApplicationObject() {
	//contructor
	this.language = "en";
	this.display = document.getElementById("appDisplay");
	this.notifier = document.getElementById("appFooterRight");
	this.navigation = document.getElementById("navigation");
	this.controls = document.getElementById("controls");
	var tab = this.newElement("li", "loggingOut", "navControl flexCenter", "Log out");
	tab.addEventListener('click', this.logOut);
	this.navigation.appendChild(tab);
	this.user = {};
	this.errorMessage = "Ok";
}

ApplicationObject.prototype.newElement = function(tag, id, className, html) {
	var e = document.createElement(tag);
	if (id) {e.id = id};
	if (className) {e.className = className};
	if (html) {e.innerHTML = html};
	return e;
}

ApplicationObject.prototype.newSelect = function(id, className, options = {}) {
	var select = this.newElement('select', id, className);
	for (var key of Object.keys(options)) {
		var option = document.createElement('option');
		option.value = key;
		option.text = options[key];
		select.add(option);
	}
	return select;
}

ApplicationObject.prototype.newTextInput = function(id, className, placeholder) {
	var input = this.newElement('input', id, className);
	input.type = "text";
	if (placeholder) {input.placeholder = placeholder;};
	return input;
}

ApplicationObject.prototype.send = function (request) {
	var xhr = new XMLHttpRequest();
	xhr.timeout = 30000;
	xhr.open('POST', "/xhr");
	xhr.send(JSON.stringify(request));
	console.log("Sending xhr", request);

	xhr.onload = function (e) {
		if (this.readyState === 4 && this.status === 200) {
			console.log(Math.round(this.responseText.length / 1024), 'Kbytes');
			var reply = JSON.parse(this.responseText);
			console.log(reply);
            switch (reply.status) {
                case "ok":
                    console.log(reply.data);
					switch (reply.type) {
						case "continue":
						case "login":
							if (reply.data == "denied") {
								app.user.authorized = false;
							} else {
								app.user.authorized = true;
								app.user.firstName = reply.data.firstName;
								app.user.lastName = reply.data.lastName;
								app.user.expiration = Math.round(reply.data.sessionExpiresIn/1000);
							}
							app.render();
							break;
						case "logout":
							window.location.assign('/')
							break;
					}
                    break;
                case "error":
                    document.getElementById('statusBar').innerHTML = reply.data;
                    break;
                default:
                    throw new Error("Unimplemented error.");
            }
		} else {
			console.error(this.statusText);
			application.showMessageBox(this.statusText, resend);
		}
	}

	xhr.onerror = function (e) {
		console.error(e);
	}

	xhr.ontimeout = function (e) {
		console.error("A request timed out.");
	}
}

ApplicationObject.prototype.render = function () {
	while (this.display.firstChild) this.display.firstChild.remove();
	// Base
	var base = app.newElement("div", null, "base flexCenterC", null);
	var header = app.newElement("div", null, "baseHeader flexCenter", null);
	var body = app.newElement("div", null, "baseBody flexCenter", null);
	var status = app.newElement("div", "statusBar", "baseStatus", null);
	base.append(header, body, status);
	//Display
	console.log(this.user);
	if (this.user.authorized) {
		if (document.location.href.includes("welcome")) {
			header.innerHTML = "Добро пожаловать";
			body.innerHTML = `Здравствуйте, ${this.user.firstName} ${this.user.lastName}`;
			body.style.backgroundColor = "#d4edf8";
			status.classList.add("flexCenter");
			app.sessionTicker = setInterval(this.updateSessionData, 1000);
		} else {
			header.innerHTML = "Вход";
			body.innerHTML = "Вход выполнен";
			status.innerHTML = `Вы авторизованы, можете <a href="/welcome.html">войти</a>`;
		}
	} else {
		header.innerHTML = "Вход";
		var id = app.newTextInput("userName", "inputbox flexCenter", "alpha");
		var pass = app.newTextInput("userPassword", "inputbox flexCenter", "omega");
		var searchButton = app.newElement("div", "inputbox loginButton", "flexCenter navControl", "Войти");
		searchButton.addEventListener('click', this.logIn);
		body.className = "baseBody flexCenterC";
		body.append(id, pass, searchButton);
		status.innerHTML = `Вы не авторизованы, но всё равно можете попробовать <a href="/welcome.html">войти</a>`;
	}
	// slotList
	// addBar
	// statusBar
	// var statusBar = app.newElement("div", "statusBar", "allWidth flexCenter");
	this.display.append(base);
}

ApplicationObject.prototype.logIn = function() {
	var id = document.getElementById('userName').value;
	var pass = document.getElementById('userPassword').value;
	if (id && pass) {
		var command = {command: "login", user: id, text: pass};
		app.send(command);
	}
}

ApplicationObject.prototype.logOut = function() {
	var command = {command: "logout", data: null};
	app.send(command);
}

ApplicationObject.prototype.updateSessionData = function() {
	document.getElementById('statusBar').innerHTML = `Ваша сессия истечёт через ${app.user.expiration} секунд`;
	app.user.expiration--;
	if (app.user.expiration < 0) {
		document.getElementById('statusBar').innerHTML = `Ваша сессия истекла.`;
		clearInterval(app.sessionTicker);
	}
}

//Init
var app = new ApplicationObject();

window.onload = function() {
	console.log("hello guys");
	app.send({command: "continue", data: null});
}