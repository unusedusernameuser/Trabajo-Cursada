# Gestor de Tareas Personales

Este proyecto es la primera entrega del trabajo práctico para la materia. Consiste en la definición del dominio de la aplicación y un servidor web básico desarrollado en **Go** que sirve una página de presentación estática en HTML.

---

## 📋 Descripción del Proyecto

Es un gestor de tareas diseñado para el día a día del usuario común. Permite crear tareas, agruparlas por categoría, y crear grupos donde varios usuarios pueden dividirse tareas en común.

### Entidades Principales

- **Tarea (`Task`)**: Elemento principal con atributos como título, descripción, fecha límite, estado (`pendiente`, `en_progreso`, `completada`), prioridad (`baja`, `media`, `alta`), y la opción de adjuntar un archivo como una imagen.
- **Categoría (`Category`)**: Clasificación para las tareas (ej. *NombreMateria*, *Recordatorio*, *Evento*, *Trabajo*).
- **Grupo (`Group`)**: Agrupación de tareas que permite colaboración entre varios usuarios (ej. *Vacaciones*, *Proyectos*).

---

## 📁 Estructura del Proyecto

```text
.
├── go.mod
├── main.go         # Servidor HTTP en Go
├── static/
│   └── index.html  # Página de presentación
└── README.md       # Instrucciones de ejecución
```

# Guía de Ejecución - Gestor de Tareas Personales

