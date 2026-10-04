const fs = require('node:fs');
fs.copyFileSync('node_modules/htmx.org/dist/htmx.min.js', 'assets/htmx.min.js');
fs.copyFileSync('web/app.js', 'assets/app.js');
fs.copyFileSync('node_modules/htmx.org/LICENSE', 'assets/htmx.LICENSE');
