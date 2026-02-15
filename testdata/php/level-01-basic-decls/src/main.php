<?php

const MAX_RETRIES = 3;

$debug = false;

function hello(): string {
    return "hello";
}

function add(int $a, int $b): int {
    return $a + $b;
}

echo hello() . "\n";
$result = add(1, 2);
