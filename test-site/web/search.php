<?php
// search.php - demonstrates reflected XSS when echoing query parameter
$db = new PDO('sqlite:' . __DIR__ . '/data.db');
$db->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);

$q = isset($_GET['q']) ? $_GET['q'] : '';
$results = [];
if ($q !== '') {
    // INSECURE: query built with string concatenation (also reflects input unsafely below)
    $sql = "SELECT id, author, comment FROM comments WHERE comment LIKE '%" . $q . "%'";
    try {
        $stmt = $db->query($sql);
        $results = $stmt->fetchAll(PDO::FETCH_ASSOC);
    } catch (Exception $e) {
        $results = [];
    }
}
?>
<!doctype html>
<html>
<head>
    <meta charset="utf-8">
    <title>Search (Reflected XSS)</title>
</head>
<body>
    <h1>Search Comments</h1>
    <form>
        <input name="q" value="<?php echo isset($_GET['q']) ? $_GET['q'] : ''; ?>">
        <button type="submit">Search</button>
    </form>

    <?php if ($q !== ''): ?>
        <h2>Results for: <?php echo $q; ?></h2>
        <?php if (count($results) === 0): ?>
            <p>No results</p>
        <?php else: ?>
            <ul>
            <?php foreach ($results as $r): ?>
                <li><strong><?php echo htmlspecialchars($r['author']); ?>:</strong> <?php echo $r['comment']; ?></li>
            <?php endforeach; ?>
            </ul>
        <?php endif; ?>
    <?php endif; ?>

    <p><a href="index.php">Back</a></p>
</body>
</html>
