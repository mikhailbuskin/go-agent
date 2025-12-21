<?php
// Simple intentionally vulnerable site index
?>
<!doctype html>
<html>
<head>
	<meta charset="utf-8">
	<title>Vulnerable Test Site</title>
</head>
<body>
	<h1>Vulnerable Test Site (SQLite)</h1>
	<ul>
		<li><a href="login.php">Login (vulnerable to SQL injection)</a></li>
		<li><a href="search.php">Search (reflects query - reflected XSS)</a></li>
		<li><a href="comments.php">Comments (stored XSS)</a></li>
		<li><a href="init_db.php">Re-initialize Database</a></li>
	</ul>
	<p><strong>Warning:</strong> This site is intentionally vulnerable for local testing only. Do not expose it to the public internet.</p>
</body>
</html>