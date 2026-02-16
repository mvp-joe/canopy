<?php

function compute(): int {
    return 42;
}

function first(): int {
    return compute();
}

function second(): int {
    return compute();
}
