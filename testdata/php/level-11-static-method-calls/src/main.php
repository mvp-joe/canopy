<?php

function compute(): void {
    $sum = Calculator::add(3, 4);
    $diff = Calculator::subtract(10, 5);
    echo $sum . " " . $diff;
}
