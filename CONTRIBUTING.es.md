# Contribuir a GripMock

**Idiomas:** [English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md) | [日本語](CONTRIBUTING.ja-JP.md) | [Deutsch](CONTRIBUTING.de.md) | Español

> Aviso: Esta página ha sido traducida automáticamente. Puede contener imprecisiones o estar incompleta. Consulte el original en inglés [`CONTRIBUTING.md`](CONTRIBUTING.md) como referencia.

¡Gracias por su interés en contribuir a GripMock! Este documento proporciona pautas para contribuir al proyecto.

## Primeros pasos

1. **Haga un fork del repositorio** y clone su fork localmente
2. **Configure su entorno de desarrollo**:
   - Instale [grpctestify](https://github.com/gripmock/grpctestify-rust) para pruebas de integración (consulte la [documentación de grpctestify](https://gripmock.github.io/grpctestify-rust/) para instrucciones de instalación)
   - Asegúrese de tener Go instalado y configurado

### Pruebas ConnectRPC

**Pruebas de cliente HTTP** (archivos `.http`) para el servidor ConnectRPC se encuentran en `examples/projects/*/connectrpc-tests.http`.

**Ejecución con httpyac:**
```bash
npx httpyac run examples/projects/greeter/connectrpc-tests/ --all
```

También puede abrir archivos `.http` en IDEs JetBrains (GoLand, IntelliJ) y hacer clic en el icono de ejecución junto a cada solicitud.

## Requisitos de prueba

### ⚠️ Reglas críticas

#### 1. Los cambios en el servidor gRPC requieren pruebas de integración

**Si cambia, añade o corrige algo relacionado con la funcionalidad del servidor gRPC, DEBE escribir pruebas de integración usando grpctestify en formato `.gctf`.**

Las pruebas de integración se encuentran en el directorio `examples/`. Ejemplo de archivo `.gctf`:

```
--- ENDPOINT ---
helloworld.Greeter/SayHello

--- REQUEST ---
{"name": "Alex"}

--- RESPONSE ---
{"message": "Hello, Alex!"}
```

**Ejecutar pruebas:**
```bash
make test              # Pruebas unitarias
grpctestify examples/  # Pruebas de integración
make lint              # Linter
```

**Dónde colocar las pruebas:**
- Pruebas de integración: `examples/projects/*/case_*.gctf`
- Pruebas unitarias: `internal/app/*_internal_test.go`

#### 2. Cada PR debe incluir pruebas

Todos los Pull Requests deben incluir pruebas apropiadas, especialmente para correcciones de errores y nuevas funcionalidades.

#### 3. Ejecutar pruebas localmente

Antes de enviar un PR, asegúrese de que todas las pruebas pasen:

**Para pruebas de integración con grpctestify:**
```bash
# Iniciar el servidor (en una terminal separada)
go run main.go examples -s examples

# Ejecutar pruebas de integración
grpctestify examples/
```

**Para pruebas unitarias:**
```bash
make test
make lint
```

## Compatibilidad hacia atrás

**Todos los cambios DEBEN ser compatibles hacia atrás** a menos que se discutan y aprueben explícitamente a través de un issue.

### Proceso de cambios disruptivos

Si necesita introducir un cambio disruptivo:

1. **Cree un Issue primero**: Abra un issue con una propuesta detallada que incluya:
   - Descripción del problema que intenta resolver
   - Por qué el cambio disruptivo es necesario
   - Ruta de migración propuesta para usuarios existentes

2. **Espere la aprobación**: No implemente cambios disruptivos sin discusión y aprobación de los mantenedores

3. **Proporcione una guía de migración**: Si se aprueba, incluya instrucciones de migración claras en su PR

## Proceso de Pull Request

### Antes de enviar

- [ ] Todas las pruebas pasan localmente
- [ ] El código sigue las pautas de estilo del proyecto (`make lint`)
- [ ] La documentación se actualiza si es necesario
- [ ] Su rama está actualizada con la rama principal

### Descripción del PR

Al crear un PR, incluya:
- Descripción de los cambios
- Tipo de cambio (corrección de errores, nueva funcionalidad, etc.)
- Información de pruebas (pruebas unitarias, pruebas de integración si hay cambios en el servidor gRPC)
- Estado de compatibilidad hacia atrás
- Issues relacionados

## Estilo de código

- Siga el formato estándar de Go: `gofmt` y `goimports`
- Ejecute el linter: `make lint`
- Use nombres de variables y funciones significativos
- Añada comentarios para funciones y tipos exportados
- Coloque el código nuevo en paquetes apropiados bajo `internal/`

## Documentación

Actualice la documentación cuando:
- Añada nuevas funcionalidades
- Cambie el comportamiento existente
- Corrija errores que afecten los flujos de trabajo de los usuarios

Ubicaciones de la documentación:
- Documentación de usuario: `docs/guide/`
- Ejemplos: directorio `examples/`
- README principal: `README.md`

## ¿Preguntas?

- Revise los issues y discussions existentes
- Abra un nuevo issue con la etiqueta `question`
- Consulte la [documentación](https://bavix.github.io/gripmock/)

## Recursos adicionales

- [Documentación del proyecto](https://bavix.github.io/gripmock/)
- [Documentación de grpctestify](https://gripmock.github.io/grpctestify-rust/)

¡Gracias por contribuir a GripMock! 🚀
