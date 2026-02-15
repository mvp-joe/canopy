<?php

require_once 'util.php';

function main(): void {
    $msg = greet("world");
    echo $msg . "\n";
    $result = add(1, 2);
    echo $result . "\n";
}

main();
