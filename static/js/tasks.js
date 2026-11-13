// Página de tareas de un usuario (tasks.html?user={id}).
// Todas las llamadas cuelgan de /users/{id}/..., así que la API solo devuelve y modifica
// lo que pertenece a ese usuario: tareas, carpetas y tags de otros usuarios nunca llegan acá.
// Las utilidades compartidas (request, sendJSON, formatDateTime...) están en common.js.

const userId = Number(new URLSearchParams(location.search).get("user"));
const BASE = `/users/${userId}`;

const STATUS_LABELS = { pendiente: "Pendiente", en_progreso: "En progreso", completada: "Completada" };
const PRIORITY_LABELS = { baja: "Baja", media: "Media", alta: "Alta" };

// Estado de la pantalla. Después de cada cambio se vuelve a pedir todo a la API
// (refresh) y se redibuja; los filtros se aplican en el cliente sobre esa lista.
const state = {
    user: null,
    tasks: [],
    folders: [],
    tags: [],
    taskTags: new Map(), // id de tarea -> tags de esa tarea
    filters: { folder: "all", tag: null, status: "", priority: "" },
    editingId: null,
};

const $ = (id) => document.getElementById(id);
const els = {
    message: $("message"),
    workspace: $("workspace"),
    folderList: $("folder-list"),
    tagList: $("tag-list"),
    taskList: $("task-list"),
    taskCount: $("task-count"),
    emptyState: $("empty-state"),
    filterSummary: $("filter-summary"),
    taskForm: $("task-form"),
    taskFolder: $("task-folder"),
    taskTags: $("task-tags"),
    editDialog: $("edit-dialog"),
    editForm: $("edit-form"),
    editError: $("edit-error"),
};

// --- Utilidades ---------------------------------------------------------

// Crea un elemento. Todo el texto entra por textContent, así que nada de lo que
// escribe el usuario se interpreta como HTML.
function h(tag, props = {}, ...children) {
    const node = document.createElement(tag);
    for (const [key, value] of Object.entries(props)) {
        if (value === undefined || value === null || value === false) continue;
        if (key === "text") node.textContent = value;
        else if (key === "dataset") Object.assign(node.dataset, value);
        else if (key in node && key !== "list") node[key] = value;
        else node.setAttribute(key, value);
    }
    node.append(...children.filter((c) => c !== null && c !== undefined && c !== false));
    return node;
}

function showMessage(text, type = "error") {
    els.message.textContent = text;
    els.message.className = `message ${type}`;
    els.message.hidden = false;
}

function clearMessage() {
    els.message.hidden = true;
    els.message.textContent = "";
}

// <input type="datetime-local"> trabaja en hora local sin zona; la API espera RFC 3339.
function toInputValue(value) {
    const date = parseDate(value);
    if (!date) return "";
    const p = (n) => String(n).padStart(2, "0");
    return `${date.getFullYear()}-${p(date.getMonth() + 1)}-${p(date.getDate())}T${p(date.getHours())}:${p(date.getMinutes())}`;
}

function fromInputValue(value) {
    return value ? new Date(value).toISOString() : null;
}

function isOverdue(task) {
    const due = parseDate(task.due_date);
    return Boolean(due) && task.task_status !== "completada" && due.getTime() < Date.now();
}

function showFatal(title, text) {
    $("fatal-title").textContent = title;
    $("fatal-text").textContent = text;
    $("fatal").hidden = false;
    els.workspace.hidden = true;
    $("user-subtitle").textContent = "";
}

// --- Carga de datos -----------------------------------------------------

async function loadAll() {
    const [tasks, folders, tags] = await Promise.all([
        request(`${BASE}/tasks`),
        request(`${BASE}/folders`),
        request(`${BASE}/tags`),
    ]);

    // Los tags de cada tarea se arman con una consulta por tag (tags/{id}/tasks).
    const perTag = await Promise.all(
        tags.map((tag) => request(`${BASE}/tags/${tag.id}/tasks`).then((list) => ({ tag, list }))),
    );
    const taskTags = new Map();
    for (const { tag, list } of perTag) {
        for (const task of list) {
            if (!taskTags.has(task.id)) taskTags.set(task.id, []);
            taskTags.get(task.id).push(tag);
        }
    }

    state.tasks = tasks;
    state.folders = folders;
    state.tags = tags;
    state.taskTags = taskTags;

    // Si se borró la carpeta o el tag que estaba filtrando, se vuelve a "todas".
    const { folder, tag } = state.filters;
    if (folder !== "all" && folder !== "none" && !folders.some((f) => String(f.id) === folder)) {
        state.filters.folder = "all";
    }
    if (tag !== null && !tags.some((t) => t.id === tag)) state.filters.tag = null;
}

async function refresh() {
    try {
        await loadAll();
        render();
    } catch (err) {
        showMessage(err.message);
    }
}

// Ejecuta una acción contra la API y, si sale bien, refresca la pantalla.
async function act(fn, okMessage) {
    clearMessage();
    try {
        await fn();
        await loadAll();
        render();
        if (okMessage) showMessage(okMessage, "success");
        return true;
    } catch (err) {
        showMessage(err.message);
        return false;
    }
}

// --- Render -------------------------------------------------------------

function tagsOf(task) {
    return state.taskTags.get(task.id) ?? [];
}

function visibleTasks() {
    const { folder, tag, status, priority } = state.filters;
    const list = state.tasks.filter((task) => {
        if (folder === "none" && task.folder_id !== null) return false;
        if (folder !== "all" && folder !== "none" && String(task.folder_id) !== folder) return false;
        if (tag !== null && !tagsOf(task).some((t) => t.id === tag)) return false;
        if (status && task.task_status !== status) return false;
        if (priority && task.task_priority !== priority) return false;
        return true;
    });
    // Las completadas van al final; dentro de cada bloque se mantiene el orden de la API (por fecha).
    return [...list.filter((t) => t.task_status !== "completada"), ...list.filter((t) => t.task_status === "completada")];
}

function renderUser() {
    const { display_name, handle, email } = state.user;
    document.title = `Tareas de ${display_name} · Gestor de Tareas`;
    $("user-title").textContent = `Tareas de ${display_name}`;
    $("user-subtitle").textContent = `@${handle} · ${email}`;
}

function navItem({ label, count, active, dataset, actions = [] }) {
    return h(
        "li",
        { className: "nav-item" },
        h(
            "button",
            { type: "button", className: `nav-btn${active ? " active" : ""}`, dataset, "aria-pressed": String(active) },
            h("span", { className: "nav-label", text: label }),
            h("span", { className: "nav-count", text: String(count) }),
        ),
        ...actions,
    );
}

function smallAction(action, id, text, label, danger = false) {
    return h("button", {
        type: "button",
        className: `btn-mini${danger ? " danger" : ""}`,
        dataset: { action, id },
        text,
        "aria-label": label,
    });
}

function renderFolders() {
    const { folder } = state.filters;
    const withoutFolder = state.tasks.filter((t) => t.folder_id === null).length;

    const items = [
        navItem({ label: "Todas", count: state.tasks.length, active: folder === "all", dataset: { folder: "all" } }),
        navItem({ label: "Sin carpeta", count: withoutFolder, active: folder === "none", dataset: { folder: "none" } }),
        ...state.folders.map((f) =>
            navItem({
                label: f.folder_name,
                count: state.tasks.filter((t) => t.folder_id === f.id).length,
                active: folder === String(f.id),
                dataset: { folder: String(f.id) },
                actions: [
                    smallAction("rename-folder", f.id, "Renombrar", `Renombrar la carpeta ${f.folder_name}`),
                    smallAction("delete-folder", f.id, "Borrar", `Borrar la carpeta ${f.folder_name}`, true),
                ],
            }),
        ),
    ];
    els.folderList.replaceChildren(...items);

    // Selector de carpeta del formulario de alta: si hay una carpeta filtrada, se propone esa.
    const previous = els.taskFolder.value;
    els.taskFolder.replaceChildren(
        h("option", { value: "", text: "Sin carpeta" }),
        ...state.folders.map((f) => h("option", { value: String(f.id), text: f.folder_name })),
    );
    const wanted = previous || (folder !== "all" && folder !== "none" ? folder : "");
    els.taskFolder.value = state.folders.some((f) => String(f.id) === wanted) ? wanted : "";
}

function renderTags() {
    const countFor = (tag) => state.tasks.filter((t) => tagsOf(t).some((x) => x.id === tag.id)).length;
    const items = state.tags.map((tag) =>
        navItem({
            label: tag.tag_name,
            count: countFor(tag),
            active: state.filters.tag === tag.id,
            dataset: { tag: String(tag.id) },
            actions: [
                smallAction("rename-tag", tag.id, "Renombrar", `Renombrar el tag ${tag.tag_name}`),
                smallAction("delete-tag", tag.id, "Borrar", `Borrar el tag ${tag.tag_name}`, true),
            ],
        }),
    );
    els.tagList.replaceChildren(
        ...(items.length ? items : [h("li", { className: "nav-empty", text: "Todavía no hay tags." })]),
    );

    // Tags elegibles en el formulario de alta (se conservan los tildados al redibujar).
    const checked = new Set([...els.taskTags.querySelectorAll("input:checked")].map((i) => i.value));
    els.taskTags.replaceChildren(
        ...(state.tags.length
            ? state.tags.map((tag) =>
                  h(
                      "label",
                      { className: "chip-check" },
                      h("input", { type: "checkbox", value: String(tag.id), checked: checked.has(String(tag.id)) }),
                      h("span", { text: tag.tag_name }),
                  ),
              )
            : [h("span", { className: "nav-empty", text: "Creá un tag desde el panel lateral para usarlo acá." })]),
    );
}

function taskItem(task) {
    const done = task.task_status === "completada";
    const overdue = isOverdue(task);
    const due = formatDateTime(task.due_date);
    const taskTagList = tagsOf(task);
    const freeTags = state.tags.filter((t) => !taskTagList.some((x) => x.id === t.id));

    const statusSelect = h(
        "select",
        { dataset: { action: "status" }, "aria-label": `Estado de ${task.title}` },
        ...Object.entries(STATUS_LABELS).map(([value, text]) => h("option", { value, text })),
    );
    statusSelect.value = task.task_status;

    const folderSelect = h(
        "select",
        { dataset: { action: "folder" }, "aria-label": `Carpeta de ${task.title}` },
        h("option", { value: "", text: "Sin carpeta" }),
        ...state.folders.map((f) => h("option", { value: String(f.id), text: f.folder_name })),
    );
    folderSelect.value = task.folder_id === null ? "" : String(task.folder_id);

    const tagRow = h(
        "div",
        { className: "chip-row" },
        ...taskTagList.map((tag) =>
            h(
                "span",
                { className: "chip" },
                tag.tag_name,
                h("button", {
                    type: "button",
                    className: "chip-x",
                    dataset: { action: "remove-tag", tag: String(tag.id) },
                    text: "×",
                    "aria-label": `Quitar el tag ${tag.tag_name}`,
                }),
            ),
        ),
    );
    if (freeTags.length) {
        const addTag = h(
            "select",
            { className: "add-tag", dataset: { action: "add-tag" }, "aria-label": `Agregar tag a ${task.title}` },
            h("option", { value: "", text: "+ Tag" }),
            ...freeTags.map((t) => h("option", { value: String(t.id), text: t.tag_name })),
        );
        tagRow.append(addTag);
    }

    return h(
        "li",
        { className: `task${done ? " done" : ""}`, dataset: { id: String(task.id) } },
        h("input", {
            type: "checkbox",
            className: "task-check",
            checked: done,
            dataset: { action: "toggle" },
            "aria-label": `Marcar como completada: ${task.title}`,
        }),
        h(
            "div",
            { className: "task-body" },
            h(
                "div",
                { className: "task-top" },
                h("span", { className: "task-title", text: task.title }),
                h("span", { className: `badge prio-${task.task_priority}`, text: PRIORITY_LABELS[task.task_priority] }),
                due ? h("span", { className: `due${overdue ? " overdue" : ""}`, text: overdue ? `Venció ${due}` : `Vence ${due}` }) : null,
            ),
            task.task_description ? h("p", { className: "task-desc", text: task.task_description }) : null,
            h("div", { className: "task-controls" }, statusSelect, folderSelect),
            tagRow,
        ),
        h(
            "div",
            { className: "task-actions" },
            h("button", { type: "button", className: "btn btn-ghost btn-sm", dataset: { action: "edit" }, text: "Editar" }),
            h("button", { type: "button", className: "btn btn-danger", dataset: { action: "delete" }, text: "Eliminar" }),
        ),
    );
}

function filterSummary() {
    const { folder, tag } = state.filters;
    const parts = [];
    if (folder === "none") parts.push("sin carpeta");
    else if (folder !== "all") parts.push(`carpeta «${state.folders.find((f) => String(f.id) === folder)?.folder_name ?? ""}»`);
    if (tag !== null) parts.push(`tag «${state.tags.find((t) => t.id === tag)?.tag_name ?? ""}»`);
    return parts.length ? `Filtrando por ${parts.join(" y ")}. Volvé a hacer clic en el filtro para quitarlo.` : "";
}

function renderTasks() {
    const list = visibleTasks();
    els.taskList.replaceChildren(...list.map(taskItem));
    els.taskCount.textContent = list.length;
    els.filterSummary.textContent = filterSummary();

    const filtered = state.tasks.length > 0 && list.length === 0;
    els.emptyState.hidden = list.length > 0;
    els.emptyState.textContent = filtered
        ? "Ninguna tarea coincide con los filtros elegidos."
        : "Todavía no hay tareas. Creá la primera con el formulario.";
}

function render() {
    renderFolders();
    renderTags();
    renderTasks();
}

// --- Eventos: formularios ------------------------------------------------

els.taskForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const data = {
        title: $("task-title").value.trim(),
        task_description: $("task-description").value.trim() || null,
        due_date: fromInputValue($("task-due").value),
        task_priority: $("task-priority").value,
        folder_id: els.taskFolder.value ? Number(els.taskFolder.value) : null,
    };
    const tagIds = [...els.taskTags.querySelectorAll("input:checked")].map((i) => i.value);

    $("task-submit").disabled = true;
    const ok = await act(async () => {
        const task = await sendJSON(`${BASE}/tasks`, "POST", data);
        for (const tagId of tagIds) {
            await request(`${BASE}/tasks/${task.id}/tags/${tagId}`, { method: "PUT" });
        }
    }, "Tarea creada.");
    $("task-submit").disabled = false;

    if (ok) {
        els.taskForm.reset();
        els.taskFolder.value = data.folder_id === null ? "" : String(data.folder_id);
        $("task-title").focus();
    }
});

$("folder-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    const input = $("folder-name");
    const ok = await act(() => sendJSON(`${BASE}/folders`, "POST", { folder_name: input.value.trim() }), "Carpeta creada.");
    if (ok) input.value = "";
});

$("tag-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    const input = $("tag-name");
    const ok = await act(() => sendJSON(`${BASE}/tags`, "POST", { tag_name: input.value.trim() }), "Tag creado.");
    if (ok) input.value = "";
});

$("filter-status").addEventListener("change", (e) => {
    state.filters.status = e.target.value;
    renderTasks();
});

$("filter-priority").addEventListener("change", (e) => {
    state.filters.priority = e.target.value;
    renderTasks();
});

// --- Eventos: carpetas y tags (delegación sobre el panel lateral) ---------

els.folderList.addEventListener("click", (event) => {
    const filterBtn = event.target.closest("[data-folder]");
    if (filterBtn) {
        const value = filterBtn.dataset.folder;
        state.filters.folder = state.filters.folder === value && value !== "all" ? "all" : value;
        render();
        return;
    }

    const btn = event.target.closest("[data-action]");
    if (!btn) return;
    const folder = state.folders.find((f) => String(f.id) === btn.dataset.id);
    if (!folder) return;

    if (btn.dataset.action === "rename-folder") {
        const name = prompt("Nuevo nombre de la carpeta:", folder.folder_name)?.trim();
        if (name && name !== folder.folder_name) {
            act(() => sendJSON(`${BASE}/folders/${folder.id}`, "PUT", { folder_name: name }), "Carpeta renombrada.");
        }
    } else if (btn.dataset.action === "delete-folder") {
        if (confirm(`¿Borrar la carpeta «${folder.folder_name}»? Sus tareas no se borran: quedan sin carpeta.`)) {
            act(() => request(`${BASE}/folders/${folder.id}`, { method: "DELETE" }), "Carpeta eliminada.");
        }
    }
});

els.tagList.addEventListener("click", (event) => {
    const filterBtn = event.target.closest("[data-tag]");
    if (filterBtn) {
        const id = Number(filterBtn.dataset.tag);
        state.filters.tag = state.filters.tag === id ? null : id;
        render();
        return;
    }

    const btn = event.target.closest("[data-action]");
    if (!btn) return;
    const tag = state.tags.find((t) => String(t.id) === btn.dataset.id);
    if (!tag) return;

    if (btn.dataset.action === "rename-tag") {
        const name = prompt("Nuevo nombre del tag:", tag.tag_name)?.trim();
        if (name && name !== tag.tag_name) {
            act(() => sendJSON(`${BASE}/tags/${tag.id}`, "PUT", { tag_name: name }), "Tag renombrado.");
        }
    } else if (btn.dataset.action === "delete-tag") {
        if (confirm(`¿Borrar el tag «${tag.tag_name}»? Se quita de todas las tareas que lo tengan.`)) {
            act(() => request(`${BASE}/tags/${tag.id}`, { method: "DELETE" }), "Tag eliminado.");
        }
    }
});

// --- Eventos: lista de tareas (delegación) --------------------------------

function taskFromEvent(event) {
    const li = event.target.closest(".task");
    return li ? state.tasks.find((t) => String(t.id) === li.dataset.id) : undefined;
}

els.taskList.addEventListener("click", (event) => {
    const btn = event.target.closest("button[data-action]");
    const task = taskFromEvent(event);
    if (!btn || !task) return;

    switch (btn.dataset.action) {
        case "delete":
            if (confirm(`¿Eliminar la tarea «${task.title}»? Esta acción no se puede deshacer.`)) {
                act(() => request(`${BASE}/tasks/${task.id}`, { method: "DELETE" }), "Tarea eliminada.");
            }
            break;
        case "edit":
            openEditor(task);
            break;
        case "remove-tag":
            act(() => request(`${BASE}/tasks/${task.id}/tags/${btn.dataset.tag}`, { method: "DELETE" }));
            break;
    }
});

els.taskList.addEventListener("change", (event) => {
    const control = event.target.closest("[data-action]");
    const task = taskFromEvent(event);
    if (!control || !task) return;

    switch (control.dataset.action) {
        case "toggle":
            act(() => sendJSON(`${BASE}/tasks/${task.id}/status`, "PUT", { task_status: control.checked ? "completada" : "pendiente" }));
            break;
        case "status":
            act(() => sendJSON(`${BASE}/tasks/${task.id}/status`, "PUT", { task_status: control.value }));
            break;
        case "folder":
            act(() => sendJSON(`${BASE}/tasks/${task.id}/folder`, "PUT", { folder_id: control.value ? Number(control.value) : null }));
            break;
        case "add-tag":
            if (control.value) act(() => request(`${BASE}/tasks/${task.id}/tags/${control.value}`, { method: "PUT" }));
            break;
    }
});

// --- Edición (diálogo modal) ----------------------------------------------

function openEditor(task) {
    state.editingId = task.id;
    $("edit-task-title").value = task.title;
    $("edit-task-description").value = task.task_description ?? "";
    $("edit-task-due").value = toInputValue(task.due_date);
    $("edit-task-priority").value = task.task_priority;
    els.editError.hidden = true;
    els.editDialog.showModal();
    $("edit-task-title").focus();
}

$("edit-cancel").addEventListener("click", () => els.editDialog.close());

els.editForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    els.editError.hidden = true;
    const body = {
        title: $("edit-task-title").value.trim(),
        task_description: $("edit-task-description").value.trim() || null,
        due_date: fromInputValue($("edit-task-due").value),
        task_priority: $("edit-task-priority").value,
    };
    try {
        await sendJSON(`${BASE}/tasks/${state.editingId}`, "PUT", body);
        els.editDialog.close();
        await refresh();
        showMessage("Tarea actualizada.", "success");
    } catch (err) {
        els.editError.textContent = err.message;
        els.editError.hidden = false;
    }
});

// --- Arranque ---------------------------------------------------------------

async function init() {
    if (!Number.isInteger(userId) || userId <= 0) {
        showFatal("Falta el usuario", "La dirección debe incluir el usuario, por ejemplo tasks.html?user=1.");
        return;
    }

    try {
        state.user = await request(`/users/${userId}`);
    } catch (err) {
        const notFound = err.message.toLowerCase().includes("no encontrado");
        showFatal(notFound ? "Usuario no encontrado" : "No se pudo abrir esta página", err.message);
        return;
    }

    renderUser();
    els.workspace.hidden = false;
    await refresh();
}

init();
