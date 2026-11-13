// Página principal del Gestor de Tareas: consume la API REST de usuarios (GET/POST/DELETE /users)
// y enlaza a la página de tareas de cada usuario (tasks.html?user={id}).
// Las utilidades compartidas (request, formatDate) están en common.js.
// El servidor Go sirve esta carpeta (static/) y la API en el mismo origen (puerto 8080),
// por eso alcanza con rutas relativas y no hace falta configurar CORS.
const API_URL = "/users";

const form = document.getElementById("user-form");
const submitBtn = document.getElementById("submit-btn");
const userList = document.getElementById("user-list");
const emptyState = document.getElementById("empty-state");
const userCount = document.getElementById("user-count");
const messageBox = document.getElementById("message");

// --- Utilidades ---------------------------------------------------------

function showMessage(text, type) {
    messageBox.textContent = text;
    messageBox.className = `message ${type}`;
    messageBox.hidden = false;
}

function clearMessage() {
    messageBox.hidden = true;
    messageBox.textContent = "";
}

// --- Render -------------------------------------------------------------

// Crea el <li> de un usuario. Se usa textContent (no innerHTML) para que
// los datos ingresados nunca se interpreten como HTML.
function createUserItem(user) {
    const li = document.createElement("li");
    li.className = "user-item";
    li.dataset.id = user.id;

    const info = document.createElement("div");
    info.className = "user-info";

    const name = document.createElement("span");
    name.className = "user-name";
    name.textContent = user.display_name + " ";

    const handle = document.createElement("span");
    handle.className = "user-handle";
    handle.textContent = "@" + user.handle;
    name.appendChild(handle);

    const meta = document.createElement("span");
    meta.className = "user-meta";
    const created = formatDate(user.created_at);
    meta.textContent = created ? `${user.email} · alta ${created}` : user.email;

    info.append(name, meta);

    const tasksLink = document.createElement("a");
    tasksLink.className = "btn btn-secondary";
    tasksLink.href = `/tasks.html?user=${encodeURIComponent(user.id)}`;
    tasksLink.textContent = "Ver tareas";
    tasksLink.setAttribute("aria-label", `Ver las tareas de ${user.display_name}`);

    const actions = document.createElement("div");
    actions.className = "user-actions";

    const deleteBtn = document.createElement("button");
    deleteBtn.type = "button";
    deleteBtn.className = "btn btn-danger";
    deleteBtn.dataset.action = "delete";
    deleteBtn.textContent = "Eliminar";
    deleteBtn.setAttribute("aria-label", `Eliminar a ${user.display_name}`);

    actions.append(tasksLink, deleteBtn);
    li.append(info, actions);
    return li;
}

function renderUsers(users) {
    userList.replaceChildren(...users.map(createUserItem));
    userCount.textContent = users.length;
    emptyState.hidden = users.length > 0;
}

// --- Comunicación con la API ---------------------------------------------

async function loadUsers() {
    try {
        const users = await request(API_URL);
        renderUsers(users);
    } catch (err) {
        showMessage(err.message, "error");
    }
}

async function createUser(event) {
    event.preventDefault();
    clearMessage();

    const data = new FormData(form);
    const payload = {
        handle: data.get("handle").trim(),
        display_name: data.get("display_name").trim(),
        email: data.get("email").trim(),
        // La API de la entrega anterior espera este campo ya hasheado,
        // pero este cliente de práctica lo envía tal cual.
        password_hash: data.get("password"),
    };

    submitBtn.disabled = true;
    try {
        await request(API_URL, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(payload),
        });
        form.reset();
        await loadUsers(); // se vuelve a pedir la lista completa
        showMessage(`Usuario @${payload.handle.toLowerCase()} creado.`, "success");
        document.getElementById("handle").focus();
    } catch (err) {
        showMessage(err.message, "error");
    } finally {
        submitBtn.disabled = false;
    }
}

async function deleteUser(id, displayName) {
    clearMessage();
    if (!confirm(`¿Eliminar a ${displayName}? Esta acción no se puede deshacer.`)) return;

    try {
        await request(`${API_URL}/${id}`, { method: "DELETE" });
        await loadUsers(); // se refresca la lista tras el borrado
        showMessage("Usuario eliminado.", "success");
    } catch (err) {
        showMessage(err.message, "error");
    }
}

// --- Eventos -------------------------------------------------------------

form.addEventListener("submit", createUser);

// Delegación de eventos: un solo listener en la lista atiende los botones
// de borrado, incluso los que se crean dinámicamente después de cargar la página.
userList.addEventListener("click", (event) => {
    const button = event.target.closest('[data-action="delete"]');
    if (!button) return;

    const item = button.closest(".user-item");
    const displayName = item.querySelector(".user-name").firstChild.textContent.trim();
    deleteUser(item.dataset.id, displayName);
});

// Al cargar la página se pide la lista inicial.
loadUsers();
