<?php
// login.php - intentionally vulnerable to SQL injection
$db = new PDO('sqlite:' . __DIR__ . '/data.db');
$db->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);

$loginResult = null;
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $user = isset($_POST['username']) ? $_POST['username'] : '';
    $pass = isset($_POST['password']) ? $_POST['password'] : '';

    // INSECURE: directly interpolate user input into SQL (vulnerable to SQLi)
    $sql = "SELECT * FROM users WHERE username = '$user' AND password = '$pass'";
    echo ($sql);
    try {
        $stmt = $db->query($sql);
        $row = $stmt->fetch(PDO::FETCH_ASSOC);
        if ($row) {
            $loginResult = "Welcome, " . htmlspecialchars($row['username']) . " (id=" . $row['id'] . ")";
        } else {
            $loginResult = "Invalid credentials";
        }
    } catch (Exception $e) {
        $loginResult = "Query error: " . htmlspecialchars($e->getMessage());
    }
}
?>
<!doctype html>
<html>
<head>
    <meta charset="utf-8">
    <title>Login (Vulnerable)</title>
</head>
<body>
    <h1>Login (vulnerable)</h1>
    <form method="post">
        <label>Username: <input name="username"></label><br>
        <label>Password: <input name="password" type="password"></label><br>
        <button type="submit">Login</button>
    </form>
    <?php if ($loginResult !== null): ?>
        <p><?php echo $loginResult; ?></p>
    <?php endif; ?>
    <p><a href="index.php">Back</a></p>
</body>
</html>
