package lazyforms

import (
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

type car struct {
	Slug    string
	Model   string
	BuiltAt time.Time
	Active  bool
	Sold    bool
}

type raceCar struct {
	Model string
}

func (c car) Persisted() bool {
	return c.Slug != ""
}

func (c car) RouteParam() string {
	return c.Slug
}

type formRouter struct{}

func (formRouter) PathForModel(model any, action string) (string, error) {
	car, _ := model.(car)
	if action == "create" {
		return "/cars", nil
	}
	return "/cars/" + car.Slug, nil
}

func (formRouter) PathFor(name string, values ...any) (string, error) {
	if name == "garage_car" && len(values) == 1 {
		return "/garage/" + values[0].(string), nil
	}
	return "", nil
}

func TestFormForRendersPartialWithActiveFormHelpers(t *testing.T) {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl":    {Data: []byte(`{{.content}}`)},
		"cars/new.html.tpl":       {Data: []byte(`{{ form_for .Car . }}`)},
		"cars/_car_form.html.tpl": {Data: []byte(`{{ text_field "Model" }}{{ date_field "BuiltAt" }}{{ checkbox_field "Active" }}{{ submit_button "Save" }}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	views.AddHelpers(Helpers(formRouter{}))
	if err := views.Cache(); err != nil {
		t.Fatal(err)
	}

	builtAt, err := time.Parse("2006-01-02", "2026-06-17")
	if err != nil {
		t.Fatal(err)
	}
	body, err := views.RenderString(lazyview.Options{
		Variables:  map[string]any{"Car": car{Model: "Roadster", BuiltAt: builtAt, Active: true}},
		Controller: "cars",
		Action:     "new",
		UseLayout:  false,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		`<form action="/cars" method="post" id="new_car" class="new_car">`,
		`name="model"`,
		`id="car_model"`,
		`value="Roadster"`,
		`type="date"`,
		`name="builtAt"`,
		`value="2026-06-17"`,
		`type="checkbox"`,
		`name="active"`,
		`checked`,
		`<button type="submit">Save</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}

func TestFormForUsesPatchForPersistedResources(t *testing.T) {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl":    {Data: []byte(`{{.content}}`)},
		"cars/edit.html.tpl":      {Data: []byte(`{{ form_for .Car . }}`)},
		"cars/_car_form.html.tpl": {Data: []byte(`{{ text_field "Model" }}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	views.AddHelpers(Helpers(formRouter{}))
	if err := views.Cache(); err != nil {
		t.Fatal(err)
	}

	body, err := views.RenderString(lazyview.Options{
		Variables:  map[string]any{"Car": car{Slug: "roadster", Model: "Roadster"}},
		Controller: "cars",
		Action:     "edit",
		UseLayout:  false,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`action="/cars/roadster"`,
		`<input type="hidden" name="_method" value="patch">`,
		`id="edit_car_roadster"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}

func TestFormForUsesUnderscoredPartialForMultiwordModels(t *testing.T) {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl":         {Data: []byte(`{{.content}}`)},
		"cars/new.html.tpl":            {Data: []byte(`{{ form_for .RaceCar . (form_action "/race-cars") }}`)},
		"cars/_race_car_form.html.tpl": {Data: []byte(`{{ text_field "Model" }}`)},
		"cars/_raceCar_form.html.tpl":  {Data: []byte(`wrong`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	views.AddHelpers(Helpers(formRouter{}))
	if err := views.Cache(); err != nil {
		t.Fatal(err)
	}

	body, err := views.RenderString(lazyview.Options{
		Variables:  map[string]any{"RaceCar": raceCar{Model: "GT"}},
		Controller: "cars",
		Action:     "new",
		UseLayout:  false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, `value="GT"`) {
		t.Fatalf("body did not render underscored partial:\n%s", body)
	}
}

func TestFormForUsesNamedRouteOption(t *testing.T) {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl":    {Data: []byte(`{{.content}}`)},
		"cars/edit.html.tpl":      {Data: []byte(`{{ form_for .Car . (form_route "garage_car" .Car.Slug) }}`)},
		"cars/_car_form.html.tpl": {Data: []byte(`{{ text_field "Model" }}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	views.AddHelpers(Helpers(formRouter{}))
	if err := views.Cache(); err != nil {
		t.Fatal(err)
	}

	body, err := views.RenderString(lazyview.Options{
		Variables:  map[string]any{"Car": car{Slug: "roadster", Model: "Roadster"}},
		Controller: "cars",
		Action:     "edit",
		UseLayout:  false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, `action="/garage/roadster"`) {
		t.Fatalf("body missing named route action:\n%s", body)
	}
}

func TestDeleteButtonForUsesDeleteOverride(t *testing.T) {
	helper := Helpers(formRouter{})["delete_button_for"].(lazyview.Helper)
	result, err := helper(nil, car{Slug: "roadster"}, "Remove")
	if err != nil {
		t.Fatal(err)
	}
	body := result.(lazyview.Fragment).Body
	for _, want := range []string{
		`action="/cars/roadster"`,
		`name="_method" value="delete"`,
		`>Remove</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}

func TestDeleteButtonForUsesNamedRouteOption(t *testing.T) {
	helper := Helpers(formRouter{})["delete_button_for"].(lazyview.Helper)
	result, err := helper(nil, car{Slug: "roadster"}, FormRoute("garage_car", "roadster"))
	if err != nil {
		t.Fatal(err)
	}
	body := result.(lazyview.Fragment).Body
	if !strings.Contains(body, `action="/garage/roadster"`) {
		t.Fatalf("body missing named route action:\n%s", body)
	}
}
