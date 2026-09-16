package main

import "fmt"

type Direccion struct {
	tag int
}

type Buton_Presionado struct {
	Caracter string
}

type Evento struct {
	tag      int
	payload2 *Buton_Presionado
	payload3 string
	payload4 float64
	payload5 bool
}

type IP struct {
	tag int
}

type Usuario struct {
	Nombre  string
	Mascota struct {
		tag      int
		payload1 string
	}
	Error struct {
		tag      int
		payload2 string
	}
}

func main() {
	var _hacha1 *Direccion = &Direccion{tag: 1}
	_ = _hacha1
	var direccion *Direccion = _hacha1
	_ = direccion
	var _hacha2 *Direccion = direccion
	_ = _hacha2
	switch _hacha2.tag {
	case 1:
		_hacha3 := fmt.Println
		_ = _hacha3
		var _hacha4 string = "izquierda"
		_ = _hacha4
		_hacha3(_hacha4)
	case 2:
		_hacha5 := fmt.Println
		_ = _hacha5
		var _hacha6 string = "arriba"
		_ = _hacha6
		_hacha5(_hacha6)
	case 3:
		_hacha7 := fmt.Println
		_ = _hacha7
		var _hacha8 string = "derecha"
		_ = _hacha8
		_hacha7(_hacha8)
	default:
		_hacha9 := fmt.Println
		_ = _hacha9
		var _hacha10 string = "abajo"
		_ = _hacha10
		_hacha9(_hacha10)
	}
	var _hacha11 string = "a"
	_ = _hacha11
	var _hacha12 *Buton_Presionado = &Buton_Presionado{Caracter: _hacha11}
	_ = _hacha12
	var boton *Buton_Presionado = _hacha12
	_ = boton
	var _hacha13 *Buton_Presionado = boton
	_ = _hacha13
	var _hacha14 *Evento = &Evento{tag: 2, payload2: _hacha13}
	_ = _hacha14
	var evento *Evento = _hacha14
	_ = evento
	var _hacha16 *Buton_Presionado = boton
	_ = _hacha16
	_hacha15 := &(_hacha16.Caracter)
	_ = _hacha15
	var _hacha17 string = "b"
	_ = _hacha17
	*_hacha15 = _hacha17
	_hacha18 := fmt.Println
	_ = _hacha18
	_hacha19 := Describir
	_ = _hacha19
	var _hacha20 *Evento = evento
	_ = _hacha20
	var _hacha21 string = _hacha19(_hacha20)
	_ = _hacha21
	_hacha18(_hacha21)
	_hacha22 := Mostrar_evento
	_ = _hacha22
	var _hacha23 *Evento = &Evento{tag: 1}
	_ = _hacha23
	_hacha22(_hacha23)
	_hacha24 := Mostrar_evento
	_ = _hacha24
	var _hacha25 *Evento = evento
	_ = _hacha25
	_hacha24(_hacha25)
	_hacha26 := Mostrar_ips
	_ = _hacha26
	_hacha26()
}

func Mostrar_evento(evento *Evento) {
	var _hacha27 *Evento = evento
	_ = _hacha27
	switch _hacha27.tag {
	case 1:
		_hacha28 := fmt.Println
		_ = _hacha28
		var _hacha29 string = "cargando pagina"
		_ = _hacha29
		_hacha28(_hacha29)
	case 2:
		e := _hacha27.payload2
		_ = e
		_hacha30 := fmt.Println
		_ = _hacha30
		var _hacha31 string = "buton presionado:"
		_ = _hacha31
		_hacha30(_hacha31)
		_hacha32 := fmt.Println
		_ = _hacha32
		var _hacha33 *Buton_Presionado = e
		_ = _hacha33
		var _hacha34 string = _hacha33.Caracter
		_ = _hacha34
		_hacha32(_hacha34)
	default:
		_hacha35 := fmt.Println
		_ = _hacha35
		var _hacha36 string = "otro evento"
		_ = _hacha36
		_hacha35(_hacha36)
	}
}

func Describir(evento *Evento) string {
	var _hacha37 string
	var _hacha38 *Evento = evento
	_ = _hacha38
	switch _hacha38.tag {
	case 1:
		var _hacha39 string = "cargando pagina"
		_ = _hacha39
		_hacha37 = _hacha39
	case 2:
		e := _hacha38.payload2
		_ = e
		_hacha40 := fmt.Println
		_ = _hacha40
		var _hacha41 string = "leyendo el boton:"
		_ = _hacha41
		_hacha40(_hacha41)
		var _hacha42 *Buton_Presionado = e
		_ = _hacha42
		var _hacha43 string = _hacha42.Caracter
		_ = _hacha43
		_hacha37 = _hacha43
	case 3:
		e := _hacha38.payload3
		_ = e
		var _hacha44 string = e
		_ = _hacha44
		_hacha37 = _hacha44
	default:
		var _hacha45 string = "otro evento"
		_ = _hacha45
		_hacha37 = _hacha45
	}
	var _hacha46 string = _hacha37
	_ = _hacha46
	var texto string = _hacha46
	_ = texto
	var _hacha47 string = texto
	_ = _hacha47
	return _hacha47
}

func A_cadena(ip *IP) string {
	var _hacha48 *IP = ip
	_ = _hacha48
	switch _hacha48.tag {
	case 1:
		var _hacha49 string = "v4"
		_ = _hacha49
		return _hacha49
	case 2:
		var _hacha50 string = "v6"
		_ = _hacha50
		return _hacha50
	default:
		panic("valor de variante inválido")
	}
}

func A_cadena2(ip *IP) string {
	var _hacha51 *IP = ip
	_ = _hacha51
	switch _hacha51.tag {
	case 1:
		var _hacha52 string = "v4"
		_ = _hacha52
		return _hacha52
	case 2:
		var _hacha53 string = "v6"
		_ = _hacha53
		return _hacha53
	default:
		panic("valor de variante inválido")
	}
}

func Mostrar_ips() {
	_hacha54 := fmt.Println
	_ = _hacha54
	_hacha55 := A_cadena
	_ = _hacha55
	var _hacha56 *IP = &IP{tag: 1}
	_ = _hacha56
	var _hacha57 string = _hacha55(_hacha56)
	_ = _hacha57
	_hacha54(_hacha57)
	_hacha58 := fmt.Println
	_ = _hacha58
	_hacha59 := A_cadena
	_ = _hacha59
	var _hacha60 *IP = &IP{tag: 2}
	_ = _hacha60
	var _hacha61 string = _hacha59(_hacha60)
	_ = _hacha61
	_hacha58(_hacha61)
}

func Abrir_archivo() struct {
	tag      int
	payload1 string
	payload2 string
} {
	var _hacha62 bool = true
	_ = _hacha62
	if _hacha62 {
		var _hacha63 struct {
			tag      int
			payload1 string
			payload2 string
		} = struct {
			tag      int
			payload1 string
			payload2 string
		}{tag: 1, payload1: "hola"}
		_ = _hacha63
		return _hacha63
	} else {
		var _hacha64 string = "error"
		_ = _hacha64
		var _hacha65 struct {
			tag      int
			payload1 string
			payload2 string
		} = struct {
			tag      int
			payload1 string
			payload2 string
		}{tag: 2, payload2: _hacha64}
		_ = _hacha65
		return _hacha65
	}
}

func Manejar() struct {
	tag      int
	payload2 string
} {
	_hacha66 := Abrir_archivo
	_ = _hacha66
	var _hacha67 struct {
		tag      int
		payload1 string
		payload2 string
	} = _hacha66()
	_ = _hacha67
	var _hacha68 string
	if _hacha67.tag == 1 {
		_hacha68 = _hacha67.payload1
	} else {
		e := _hacha67.payload2
		_ = e
		var _hacha69 string = "hola"
		_ = _hacha69
		_hacha68 = _hacha69
	}
	var _hacha70 string = _hacha68
	_ = _hacha70
	var texto string = _hacha70
	_ = texto
	var _hacha71 struct {
		tag      int
		payload2 string
	} = struct {
		tag      int
		payload2 string
	}{tag: 1}
	_ = _hacha71
	return _hacha71
}

func Nulable() {
	var _hacha72 struct {
		tag      int
		payload1 string
	} = struct {
		tag      int
		payload1 string
	}{tag: 1, payload1: "hola"}
	_ = _hacha72
	var _hacha73 struct {
		tag      int
		payload2 string
	} = struct {
		tag      int
		payload2 string
	}{tag: 1}
	_ = _hacha73
	var _hacha74 *Usuario = &Usuario{Mascota: _hacha72, Error: _hacha73}
	_ = _hacha74
	var usuario *Usuario = _hacha74
	_ = usuario
	var _hacha75 *Usuario = usuario
	_ = _hacha75
	var _hacha76 struct {
		tag      int
		payload1 string
	} = _hacha75.Mascota
	_ = _hacha76
	var mascota struct {
		tag      int
		payload1 string
	} = _hacha76
	_ = mascota
	var _hacha77 struct {
		tag      int
		payload2 string
	} = struct {
		tag      int
		payload2 string
	}{tag: 1}
	_ = _hacha77
	var _hacha78 *Usuario = &Usuario{Error: _hacha77}
	_ = _hacha78
	var _hacha79 struct {
		tag      int
		payload1 *Usuario
	} = struct {
		tag      int
		payload1 *Usuario
	}{tag: 1, payload1: _hacha78}
	_ = _hacha79
	var talvez_usuario struct {
		tag      int
		payload1 *Usuario
	} = _hacha79
	_ = talvez_usuario
	var _hacha80 string
	{
		var _hacha81 struct {
			tag      int
			payload1 string
		} = mascota
		_ = _hacha81
		if _hacha81.tag == 1 {
			m := _hacha81.payload1
			_ = m
			var _hacha82 string = m
			_ = _hacha82
			_hacha80 = _hacha82
		} else {
			var _hacha83 string = ""
			_ = _hacha83
			_hacha80 = _hacha83
		}
	}
	var _hacha84 string = _hacha80
	_ = _hacha84
	var c string = _hacha84
	_ = c
	var _hacha85 struct {
		tag      int
		payload1 string
	} = mascota
	_ = _hacha85
	var _hacha86 string
	if _hacha85.tag == 1 {
		_hacha86 = _hacha85.payload1
	} else {
		var _hacha87 string = ""
		_ = _hacha87
		_hacha86 = _hacha87
	}
	var _hacha88 string = _hacha86
	_ = _hacha88
	var c2 string = _hacha88
	_ = c2
}
