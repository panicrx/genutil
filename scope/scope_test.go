package scope_test

import (
	"fmt"
	"testing"
	"unicode/utf8"

	"github.com/panicrx/genutil/scope"
)

func TestExported(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"x",
			args{"x"},
			"X",
		},
		{
			"X",
			args{"X"},
			"X",
		},
		{
			"_X",
			args{"_X"},
			"X",
		},
		{
			"漢字a",
			args{"漢字a"},
			"A",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scope.Exported(tt.args.name); got != tt.want {
				t.Errorf("Exported() = %v, want %v", got, tt.want)
			}
		})
	}

	failureCases := []struct {
		name string
		args args
	}{
		{
			"_",
			args{"_"},
		},
		{
			"",
			args{""},
		},
		{
			"_1",
			args{"_1"},
		},
		{
			"123",
			args{"123"},
		},
		{
			"RuneError",
			args{fmt.Sprintf("%c", utf8.RuneError)},
		},
		{
			"漢字",
			args{"漢字"},
		},
	}

	for _, tt := range failureCases {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				recover()
			}()
			scope.Exported(tt.args.name)
			t.Errorf("Exported() expected panic but did not")
		})
	}
}

func TestUnexported(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"x",
			args{"x"},
			"x",
		},
		{
			"X",
			args{"X"},
			"x",
		},
		{
			"_X",
			args{"_X"},
			"_X",
		},
		{
			"_",
			args{"_"},
			"_",
		},
		{
			"_1",
			args{"_1"},
			"_1",
		},
		{
			"漢字a",
			args{"漢字a"},
			"漢字a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scope.Unexported(tt.args.name); got != tt.want {
				t.Errorf("Unexported() = %v, want %v", got, tt.want)
			}
		})
	}

	failureCases := []struct {
		name string
		args args
	}{
		{
			"",
			args{""},
		},
		{
			"123",
			args{"123"},
		},
		{
			"RuneError",
			args{fmt.Sprintf("%c", utf8.RuneError)},
		},
	}

	for _, tt := range failureCases {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				recover()
			}()
			scope.Unexported(tt.args.name)
			t.Errorf("Unexported() expected panic but did not")
		})
	}
}

func TestScope_Claim(t *testing.T) {
	t.Run("reserved name handling", func(t *testing.T) {
		tests := []struct {
			name string
			want string
		}{
			{
				"",
				"v",
			},
			{
				"panic",
				"_panic",
			},
			{
				"func",
				"_func",
			},
			{
				"123",
				"_123",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				s := &scope.Scope{}
				if got := s.Claim(tt.name); got != tt.want {
					t.Errorf("Claim() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("unique name handling", func(t *testing.T) {
		if testing.Short() {
			t.Skip()
			return
		}

		s := &scope.Scope{}
		m := map[string]bool{}
		for i := 0; i < 1000; i++ {
			g := s.Claim("v")
			if m[g] {
				t.Errorf("Claim: returned non-unique value %v", g)
			}
			m[g] = true
		}

		defer func() {
			recover()
		}()
		s.Claim("v")
		t.Errorf("Claim() expected panic but did not")
	})
}

func TestScope_ClaimGlobal(t *testing.T) {
	t.Run("reserved name handling", func(t *testing.T) {
		tests := []struct {
			name     string
			want     string
			wantNext string
		}{
			{
				"",
				"v",
				"v0",
			},
			{
				"panic",
				"_panic",
				"_panic0",
			},
			{
				"func",
				"_func",
				"_func0",
			},
			{
				"123",
				"_123",
				"_1230",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				g0 := &scope.Scope{}
				if got := g0.Claim(tt.name); got != tt.want {
					t.Errorf("g0.Claim() = %v, want %v", got, tt.want)
				}

				s0 := g0.New()
				if got := s0.ClaimGlobal(tt.name); got != tt.wantNext {
					t.Errorf("s0.ClaimGlobal() = %v, wantNext %v", got, tt.wantNext)
				}

				g1 := &scope.Scope{}
				s1 := g1.New()
				if got := s1.ClaimGlobal(tt.name); got != tt.want {
					t.Errorf("s1.ClaimGlobal() = %v, want %v", got, tt.want)
				}

				if got := g1.Claim(tt.name); got != tt.wantNext {
					t.Errorf("g1.Claim() = %v, wantNext %v", got, tt.wantNext)
				}
			})
		}
	})

	t.Run("unique name handling", func(t *testing.T) {
		if testing.Short() {
			t.Skip()
			return
		}

		s := &scope.Scope{}
		m := map[string]bool{}
		for i := 0; i < 1000; i++ {
			g := s.ClaimGlobal("v")
			if m[g] {
				t.Errorf("ClaimGlobal: returned non-unique value %v", g)
			}
			m[g] = true
		}

		defer func() {
			recover()
		}()
		s.ClaimGlobal("v")
		t.Errorf("ClaimGlobal() expected panic but did not")
	})
}

func TestScope_Suggest(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			"a",
			"a",
		},
		{
			"aPerson",
			"ap",
		},
		{
			"FxUxNxC",
			"_func",
		},
		{
			"HTTP",
			"h",
		},
		{
			"_",
			"v",
		},
		{
			fmt.Sprintf("%c", utf8.RuneError),
			"v",
		},
		{
			fmt.Sprintf("_%c", utf8.RuneError),
			"v",
		},
		{
			fmt.Sprintf("_%c", utf8.RuneError),
			"v",
		},
		{
			fmt.Sprintf("AbCd%c", utf8.RuneError),
			"a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			s := &scope.Scope{}
			if got := s.Suggest(tt.input); got != tt.want {
				t.Errorf("Suggest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Run("zero", func(t *testing.T) {
		s := scope.New()

		if s.Claim("panic") != "_panic" {
			t.Errorf("New() not using default safe name func")
		}

		if s.Claim("panic") != "_panic0" {
			t.Errorf("New() not using default unique name func")
		}

		if s.Suggest("TablePerson") != "tp" {
			t.Errorf("New() not using default suggest name func")
		}
	})

	t.Run("custom safe name", func(t *testing.T) {
		s := scope.New(scope.WithSafeNameFunc(func(name string) string {
			return "safe"
		}))

		if s.Claim("panic") != "safe" {
			t.Errorf("New() not using overridden safe name func")
		}

		t.Run("child with override", func(t *testing.T) {
			c := s.New(scope.WithSafeNameFunc(func(name string) string {
				return "override"
			}))

			if c.Claim("panic") != "override" {
				t.Errorf("New() not using overridden safe name func")
			}
		})

		t.Run("child without override", func(t *testing.T) {
			c := s.New()

			if c.Claim("panic") != "safe" {
				t.Errorf("New() not using fallback safe name func")
			}
		})
	})

	t.Run("custom unique name", func(t *testing.T) {
		s := scope.New(scope.WithUniqueNameFunc(func(s *scope.Scope, name string, recursive bool) string {
			return "unique"
		}))

		s.Claim("panic")
		if s.Claim("panic") != "unique" {
			t.Errorf("New() not using overridden unique name func")
		}

		t.Run("child with override", func(t *testing.T) {
			c := s.New(scope.WithUniqueNameFunc(func(s *scope.Scope, name string, recursive bool) string {
				return "override"
			}))

			c.Claim("panic")
			if c.Claim("panic") != "override" {
				t.Errorf("New() not using overridden unique name func")
			}
		})

		t.Run("child without override", func(t *testing.T) {
			c := s.New()

			c.Claim("panic")
			if c.Claim("panic") != "unique" {
				t.Errorf("New() not using fallback unique name func")
			}
		})
	})

	t.Run("custom suggest name", func(t *testing.T) {
		s := scope.New(scope.WithSuggestVarNameFunc(func(name string) string {
			return "suggest"
		}))

		if s.Suggest("apple") != "suggest" {
			t.Errorf("New() not using overridden suggest name func")
		}

		t.Run("child with override", func(t *testing.T) {
			c := s.New(scope.WithSuggestVarNameFunc(func(input string) string {
				return "override"
			}))

			if c.Suggest("panic") != "override" {
				t.Errorf("New() not using overridden suggest name func")
			}
		})

		t.Run("child without override", func(t *testing.T) {
			c := s.New()

			if c.Suggest("panic") != "suggest" {
				t.Errorf("New() not using fallback suggest name func")
			}
		})
	})
}
