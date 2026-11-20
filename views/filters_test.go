package views

import (
	"net/url"
	"testing"
)

func TestFiltersFromValues(t *testing.T) {
	v := url.Values{
		"folder":   {"007"},
		"tag":      {"3"},
		"status":   {"completada"},
		"priority": {"urgentisima"}, // inválida: se descarta
	}
	got := FiltersFromValues(v)
	want := TaskFilters{Folder: "7", Tag: 3, Status: "completada"}
	if got != want {
		t.Errorf("FiltersFromValues = %+v, se esperaba %+v", got, want)
	}

	if got := FiltersFromValues(url.Values{"folder": {"abc"}, "tag": {"-1"}}); got != (TaskFilters{Folder: "all"}) {
		t.Errorf("filtros inválidos deberían descartarse, se obtuvo %+v", got)
	}
	if got := FiltersFromValues(url.Values{"folder": {"none"}}); got.Folder != "none" {
		t.Errorf("folder=none: se obtuvo %+v", got)
	}
}

func TestTasksURL(t *testing.T) {
	if got := TasksURL(5, TaskFilters{Folder: "all"}); got != "/users/5/tasks" {
		t.Errorf("sin filtros: %q", got)
	}
	got := TasksURL(5, TaskFilters{Folder: "2", Tag: 3, Status: "pendiente", Priority: "alta"})
	want := "/users/5/tasks?folder=2&priority=alta&status=pendiente&tag=3"
	if got != want {
		t.Errorf("con filtros: %q, se esperaba %q", got, want)
	}
}

func TestFolderAndTagToggle(t *testing.T) {
	d := TasksPageData{UserID: 1, Filters: TaskFilters{Folder: "4", Tag: 9}}
	if got := folderToggleURL(d, "4"); got != "/users/1/tasks?tag=9" {
		t.Errorf("clic en la carpeta activa debe quitar el filtro: %q", got)
	}
	if got := folderToggleURL(d, "none"); got != "/users/1/tasks?folder=none&tag=9" {
		t.Errorf("clic en otra carpeta: %q", got)
	}
	if got := tagToggleURL(d, 9); got != "/users/1/tasks?folder=4" {
		t.Errorf("clic en el tag activo debe quitar el filtro: %q", got)
	}
}
