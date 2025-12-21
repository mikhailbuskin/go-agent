<?php
// init_db.php
// Creates a simple SQLite database with users and comments for the vulnerable demo.
$dbFile = __DIR__ . '/data.db';
if (file_exists($dbFile)) {
    unlink($dbFile);
}

try {
    $db = new PDO('sqlite:' . $dbFile);
    $db->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);

    $db->exec("CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT, password TEXT);");
    $db->exec("CREATE TABLE comments (id INTEGER PRIMARY KEY AUTOINCREMENT, author TEXT, comment TEXT);");

    // Sample users (passwords stored in plain text intentionally)
    $db->exec("INSERT INTO users (username, password) VALUES ('alice', 'password123');");
    $db->exec("INSERT INTO users (username, password) VALUES ('bob', 'hunter2');");

    // Sample comments
    $db->exec("INSERT INTO comments (author, comment) VALUES ('alice', 'Hello world');");
    $db->exec("INSERT INTO comments (author, comment) VALUES ('bob', 'Nice site!');");
    // Ensure the DB file is writable by the web server (for local testing only)
    if (file_exists($dbFile)) {
        @chmod($dbFile, 0666);
    }

    echo "Database initialized at: " . htmlspecialchars($dbFile);
} catch (Exception $e) {
    echo "Error initializing DB: " . htmlspecialchars($e->getMessage());
}
