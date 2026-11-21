// Package views contiene los componentes templ que generan el HTML de la aplicación.
package views

import (
	"database/sql"
	"net/url"
	"os"
	"slices"
	"strconv"
	"time"
	_ "time/tzdata" // incluye la base de zonas horarias en el binario (la imagen alpine no la trae)

	"github.com/a-h/templ"
)

// ---- Usuarios ----------------------------------------------------------------

// UserFormValues son los datos con los que se dibuja el formulario de alta.
// Si el servidor rechaza el envío, se vuelve a mostrar con lo que escribió la
// persona (salvo la contraseña) y el mensaje de error.
type UserFormValues struct {
	Handle      string
	DisplayName string
	Email       string
	Error       string
	// OOB marca el formulario con hx-swap-oob: así la respuesta de un alta exitosa puede
	// reemplazarlo por uno vacío mientras el target principal recibe la lista.
	OOB bool
}

// formatDate muestra una fecha dd/mm/aaaa o vacío si la columna es NULL.
func formatDate(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("02/01/2006")
}

// userDeleteURL es la ruta DELETE /users/{id} que usa hx-delete.
func userDeleteURL(id int32) string {
	return "/users/" + strconv.Itoa(int(id))
}

// userTasksURL es la página de tareas de un usuario.
func userTasksURL(id int32) string {
	return userPath(id, "/tasks")
}

// ---- Zona horaria y fechas ---------------------------------------------------

// Location es la zona horaria con la que se muestran y se interpretan las fechas
// (las columnas son timestamptz). Se puede cambiar con la variable APP_TIMEZONE.
var Location = loadLocation()

func loadLocation() *time.Location {
	name := os.Getenv("APP_TIMEZONE")
	if name == "" {
		name = "America/Argentina/Buenos_Aires"
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

// FormatDateTime muestra una fecha y hora dd/mm/aaaa hh:mm (vacío si es NULL).
func FormatDateTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.In(Location).Format("02/01/2006 15:04")
}

// InputDateTime da el valor que espera <input type="datetime-local"> (vacío si es NULL).
func InputDateTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.In(Location).Format("2006-01-02T15:04")
}

// ---- Tareas: opciones válidas y filtros --------------------------------------

// Valores permitidos por el schema.
var (
	Statuses   = []string{"pendiente", "en_progreso", "completada"}
	Priorities = []string{"baja", "media", "alta"}
)

// TaskFilters son los filtros de la lista de tareas. Viajan en la query string
// (GET /users/{id}/tasks?folder=3&tag=2&status=pendiente&priority=alta), así que
// cada combinación tiene su propia URL.
type TaskFilters struct {
	Folder   string // "all" (todas), "none" (sin carpeta) o el id de una carpeta
	Tag      int32  // 0 = sin filtro por tag
	Status   string // "" = todos
	Priority string // "" = todas
}

// FiltersFromValues lee los filtros de una query string, descartando lo que no sea válido.
func FiltersFromValues(v url.Values) TaskFilters {
	f := TaskFilters{Folder: "all"}

	if folder := v.Get("folder"); folder == "none" {
		f.Folder = "none"
	} else if id, err := strconv.ParseInt(folder, 10, 32); err == nil && id > 0 {
		f.Folder = strconv.FormatInt(id, 10)
	}
	if id, err := strconv.ParseInt(v.Get("tag"), 10, 32); err == nil && id > 0 {
		f.Tag = int32(id)
	}
	if s := v.Get("status"); slices.Contains(Statuses, s) {
		f.Status = s
	}
	if p := v.Get("priority"); slices.Contains(Priorities, p) {
		f.Priority = p
	}
	return f
}

// Query devuelve los filtros activos como query string ("" si no hay ninguno).
func (f TaskFilters) Query() string {
	v := url.Values{}
	if f.Folder != "" && f.Folder != "all" {
		v.Set("folder", f.Folder)
	}
	if f.Tag != 0 {
		v.Set("tag", strconv.Itoa(int(f.Tag)))
	}
	if f.Status != "" {
		v.Set("status", f.Status)
	}
	if f.Priority != "" {
		v.Set("priority", f.Priority)
	}
	return v.Encode()
}

// Active indica si hay algún filtro aplicado.
func (f TaskFilters) Active() bool {
	return f.Query() != ""
}

// TasksURL es la URL de la página de tareas de un usuario con esos filtros.
func TasksURL(userID int32, f TaskFilters) string {
	u := userTasksURL(userID)
	if q := f.Query(); q != "" {
		u += "?" + q
	}
	return u
}

// ---- Tareas: modelos de las páginas ------------------------------------------

// NamedRef es un id con su nombre (carpeta o tag) y la cantidad de tareas que tiene.
type NamedRef struct {
	ID    int32
	Name  string
	Count int
}

// TaskRow es una tarea lista para dibujar.
type TaskRow struct {
	ID          int32
	Title       string
	Description string
	Status      string
	Priority    string
	DueText     string // fecha límite ya formateada ("" si no tiene)
	Overdue     bool
	FolderID    int32      // 0 = sin carpeta
	Tags        []NamedRef // tags que ya tiene
	FreeTags    []NamedRef // tags del usuario que todavía no tiene
}

// TaskFormValues son los datos del formulario de alta de tareas. Si el servidor lo rechaza,
// se vuelve a mostrar con lo que escribió la persona.
type TaskFormValues struct {
	Title       string
	Description string
	Due         string // valor de <input type="datetime-local">
	Priority    string
	FolderID    int32 // 0 = sin carpeta
	TagIDs      []int32
}

// TasksPageData son todos los datos de la página de tareas de un usuario.
type TasksPageData struct {
	UserID        int32
	DisplayName   string
	Handle        string
	Email         string
	Filters       TaskFilters
	FilterSummary string
	EmptyText     string
	TotalTasks    int
	WithoutFolder int
	Folders       []NamedRef
	Tags          []NamedRef
	Tasks         []TaskRow
	NewTask       TaskFormValues
	Error         string
}

// TaskEditData son los datos de la página de edición de una tarea.
type TaskEditData struct {
	UserID      int32
	TaskID      int32
	Title       string
	Description string
	Due         string
	Priority    string
	Filters     TaskFilters
	Error       string
}

// ---- Helpers usados por las plantillas ---------------------------------------

func idStr(id int32) string { return strconv.Itoa(int(id)) }

func num(n int) string { return strconv.Itoa(n) }

// userPath arma una ruta que cuelga de /users/{id}.
func userPath(userID int32, rest string) string {
	return "/users/" + idStr(userID) + rest
}

func hasID(ids []int32, id int32) bool { return slices.Contains(ids, id) }

// folderToggleURL: al hacer clic en la carpeta activa se quita el filtro.
func folderToggleURL(d TasksPageData, folder string) string {
	f := d.Filters
	if f.Folder == folder && folder != "all" {
		f.Folder = "all"
	} else {
		f.Folder = folder
	}
	return TasksURL(d.UserID, f)
}

// tagToggleURL: al hacer clic en el tag activo se quita el filtro.
func tagToggleURL(d TasksPageData, tagID int32) string {
	f := d.Filters
	if f.Tag == tagID {
		f.Tag = 0
	} else {
		f.Tag = tagID
	}
	return TasksURL(d.UserID, f)
}

func editURL(d TasksPageData, t TaskRow) string {
	u := userPath(d.UserID, "/tasks/"+idStr(t.ID)+"/edit")
	if q := d.Filters.Query(); q != "" {
		u += "?" + q
	}
	return u
}

func taskClass(t TaskRow) string {
	if t.Status == "completada" {
		return "task done"
	}
	return "task"
}

func dueClass(t TaskRow) string {
	if t.Overdue {
		return "overdue"
	}
	return ""
}

func dueLabel(t TaskRow) string {
	if t.Overdue {
		return "Venció " + t.DueText
	}
	return "Vence " + t.DueText
}

// toggleStatus es el estado al que pasa la tarea con el botón rápido.
func toggleStatus(t TaskRow) string {
	if t.Status == "completada" {
		return "pendiente"
	}
	return "completada"
}

func toggleLabel(t TaskRow) string {
	if t.Status == "completada" {
		return "Reabrir"
	}
	return "Completar"
}

func statusLabel(s string) string {
	switch s {
	case "en_progreso":
		return "En progreso"
	case "completada":
		return "Completada"
	}
	return "Pendiente"
}

func priorityLabel(p string) string {
	switch p {
	case "baja":
		return "Baja"
	case "alta":
		return "Alta"
	}
	return "Media"
}

// oobAttrs devuelve hx-swap-oob="true" cuando el componente se envía fuera de banda.
func oobAttrs(oob bool) templ.Attributes {
	if !oob {
		return nil
	}
	return templ.Attributes{"hx-swap-oob": "true"}
}
