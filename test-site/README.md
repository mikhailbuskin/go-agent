# Vulnerable Test Site (SQLite)

This `test-site` folder contains a deliberately insecure PHP website used for local security training and testing. It includes examples of:

- SQL Injection (in `login.php`) — insecure string concatenation used in SQL queries.
- Reflected XSS (in `search.php`) — user-supplied input is reflected into the page without proper escaping.
- Stored XSS (in `comments.php`) — comments are stored and later rendered unsafely.

Safety notes:
- This site is intentionally vulnerable. Only run locally on an isolated machine or inside a private network.
- Do not expose this site to the public internet.

How to run (Windows):

1. Open a command prompt in `test-site`.
2. Run `start.bat` which will initialize the SQLite DB and start a PHP-Apache Docker container serving the `web` folder on port `8082`.

Example commands (PowerShell / cmd):

    cd test-site
    start.bat

Access the site at: http://localhost:8082/

If you need to reinitialize the database, visit `http://localhost:8082/init_db.php` or re-run the `start.bat` which runs the init step.

Use this project responsibly and only for learning or testing in controlled environments.
