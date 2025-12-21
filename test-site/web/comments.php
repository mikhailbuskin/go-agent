<?php
// comments.php - demonstrates stored XSS (comments are stored and displayed unsanitized)
$db = new PDO('sqlite:' . __DIR__ . '/data.db');
$db->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $author = isset($_POST['author']) ? $_POST['author'] : 'anon';
    $comment = isset($_POST['comment']) ? $_POST['comment'] : '';

    // INSECURE: directly insert user content without sanitization
    $sql = "INSERT INTO comments (author, comment) VALUES ('" . str_replace("'","''",$author) . "', '" . str_replace("'","''",$comment) . "')";
    try {
        $db->exec($sql);
    } catch (Exception $e) {
        // ignore
        var_dump($e);
    }
    header('Location: comments.php');
    exit;
}

$comments = $db->query('SELECT * FROM comments ORDER BY id DESC')->fetchAll(PDO::FETCH_ASSOC);
?>
<!doctype html>
<html>
<head>
    <meta charset="utf-8">
    <title>Comments (Stored XSS)</title>
</head>
<body>
    <h1>Comments</h1>
    <form method="post">
        <label>Name: <input name="author"></label><br>
        <label>Comment: <textarea name="comment"></textarea></label><br>
        <button type="submit">Post Comment</button>
    </form>

    <h2>Recent Comments</h2>
    <ul>
    <?php foreach ($comments as $c): ?>
        <li><strong><?php echo htmlspecialchars($c['author']); ?>:</strong> <?php echo $c['comment']; ?></li>
    <?php endforeach; ?>
    </ul>

    <p><a href="index.php">Back</a></p>
</body>
</html>
