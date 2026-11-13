// Utilidades compartidas por las páginas del cliente (index.html y tasks.html).
// Se carga antes que app.js / tasks.js, así que sus funciones quedan disponibles globalmente.

// Hace el fetch y devuelve el JSON (o null si la respuesta es 204).
// Si el servidor responde con error, lanza un Error con el mensaje de la API ({"error": "..."}).
async function request(url, options) {
    let response;
    try {
        response = await fetch(url, options);
    } catch {
        throw new Error("No se pudo conectar con el servidor. ¿Está corriendo la API?");
    }

    if (!response.ok) {
        let message = `Error ${response.status}`;
        try {
            const body = await response.json();
            if (body.error) message = body.error;
        } catch {
            // la respuesta no era JSON: dejamos el mensaje genérico
        }
        throw new Error(message);
    }

    return response.status === 204 ? null : response.json();
}

// Envía un cuerpo JSON con el método indicado (POST, PUT...).
function sendJSON(url, method, body) {
    return request(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
    });
}

// Convierte el valor que llega de la API en un Date válido (o null).
// Acepta un string ISO o el objeto que serializa sql.NullTime de Go: {"Time": "...", "Valid": true}.
function parseDate(value) {
    const isoString = value && typeof value === "object" ? (value.Valid ? value.Time : null) : value;
    if (!isoString) return null;
    const date = new Date(isoString);
    return Number.isNaN(date.getTime()) ? null : date;
}

function formatDate(value) {
    const date = parseDate(value);
    if (!date) return "";
    return date.toLocaleDateString("es-AR", {
        day: "numeric",
        month: "short",
        year: "numeric",
    });
}

function formatDateTime(value) {
    const date = parseDate(value);
    if (!date) return "";
    return date.toLocaleString("es-AR", {
        day: "numeric",
        month: "short",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
    });
}
