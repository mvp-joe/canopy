<?php

class MathHelper {
    public const PI = 3.14159;

    public static function add(int $a, int $b): int {
        return $a + $b;
    }

    public static function multiply(int $a, int $b): int {
        return $a * $b;
    }
}

function calculate(): void {
    $sum = MathHelper::add(3, 4);
    $product = MathHelper::multiply(5, 6);
    echo $sum . " " . $product . "\n";
}
