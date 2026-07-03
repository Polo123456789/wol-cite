# wol-cite

Utilidad de linea de comandos para convertir una cita biblica en texto tomado
desde WOL (`wol.jw.org`).

Este README no intenta presentar el proyecto como producto. Su proposito es
dejar claro cual es la intencion del programa y que comportamiento debe
mantenerse si la estructura de la pagina de WOL cambia en el futuro.

## Intencion

El programa recibe una cita en formato:

```text
Libro Capitulo:Versiculos
```

Luego descarga el capitulo correspondiente desde WOL y devuelve solo los
versiculos solicitados, incluyendo el numero de cada versiculo y removiendo los
marcadores de notas al pie y referencias marginales.

La primera integracion prevista es con nvim, por eso la cita puede llegar por
argumentos o por `stdin`.

## Uso

```sh
go run . Genesis 1:1
go run . Genesis 1:1-3
go run . Genesis 1:1, 2, 5
printf 'Jn 11:11-15' | go run .
go run . --json Jn 11:11-15
```

La salida normal sigue estas reglas:

- Si la cita pide un solo versiculo o un rango, concatena los versiculos en una
  sola linea.
- Si la cita pide una lista separada por comas, imprime un versiculo por linea.

Con `--json`, la salida contiene:

- `reference`: cita normalizada.
- `source_url`: URL consultada.
- `verses`: lista de objetos con `number` y `text`.
- `text`: el mismo texto que se imprimiria en modo normal.

Los errores se escriben en `stderr` y el programa termina con exit code `1`.

## Contrato funcional

Esto es lo que debe seguir funcionando aunque cambie el HTML de WOL:

- Aceptar citas como argumentos: `wol-cite Genesis 1:1`.
- Aceptar citas por `stdin` cuando no hay argumentos.
- Soportar versiculo unico: `Genesis 1:1`.
- Soportar listas: `Genesis 1:1, 2, 5`.
- Soportar rangos: `Genesis 1:1-3`.
- Soportar abreviaturas de libros, por ejemplo `Gen`, `Ge` y `Jn`.
- Devolver el texto de WOL con el numero de versiculo incluido.
- Quitar de la cita final los simbolos de notas al pie y referencias
  marginales, por ejemplo `*` y `+`.
- Mantener el modo JSON para integraciones con otros programas.

No se pretende soportar citas multicapitulo en la version actual.

## Formato de WOL esperado actualmente

La URL de un capitulo usa este formato:

```text
https://wol.jw.org/es/wol/b/r4/lp-s/nwtsty/{bookID}/{chapter}
```

Ejemplo:

```text
https://wol.jw.org/es/wol/b/r4/lp-s/nwtsty/43/11
```

Al momento de escribir esto, la pagina de WOL tiene estas caracteristicas:

- Cada versiculo esta en un nodo `span` con clase `v`.
- El `id` del versiculo tiene forma `v{bookID}-{chapter}-{verse}-...`.
- El numero del versiculo esta dentro del mismo `span.v`, normalmente en un
  enlace con clase `vl` o `cl`.
- Las notas al pie son enlaces `a.fn`.
- Las referencias marginales son enlaces `a.b`.

La extraccion actual esta en `extractVerses` y `verseText` dentro de
`main.go`. Si WOL cambia el HTML, esas funciones son el primer lugar que hay que
revisar.

## Guia para mantenimiento por agente

Si en el futuro el programa deja de encontrar versiculos o empieza a incluir
marcadores no deseados:

1. Reproducir el fallo con una cita real, por ejemplo:

   ```sh
   go run . Jn 11:11-15
   ```

2. Descargar o inspeccionar el HTML actual del capitulo afectado.

3. Confirmar donde estan ahora:

   - El contenedor de cada versiculo.
   - El numero de versiculo.
   - El texto real del versiculo.
   - Los enlaces o marcas de notas al pie.
   - Los enlaces o marcas de referencias marginales.

4. Actualizar la extraccion manteniendo el contrato funcional de este README.

5. Actualizar o ampliar los fixtures de `main_test.go` para cubrir el nuevo
   HTML.

6. Ejecutar:

   ```sh
   go test ./...
   ```

7. Verificar al menos una cita real contra WOL, idealmente una que tenga notas
   al pie y referencias marginales.

El parser de citas y el mapa de libros no deberian cambiar solo porque cambio
el HTML de WOL. Cambiarlos solamente si falla una entrada valida o si se decide
ampliar explicitamente el formato soportado.

## Dependencia HTML

El parser HTML usado por el programa es:

```text
golang.org/x/net/html
```

No se usa `goquery`. Si se necesita ajustar la extraccion, conservar esta
dependencia salvo que haya una razon concreta para cambiarla.
