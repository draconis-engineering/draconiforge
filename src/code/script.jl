# A simple math function defined in a single line
square(x) = x^2

# Multi-line function with explicit type annotations
function greet_user(name::String, age::Int)
    println("Hello $name, you are $age years old.")
    return age + 1
end

# Multiple dispatch: Same function name, different argument types
function describe_item(x::String)
    println("This is text: $x")
end

function describe_item(x::Number)
    println("This is a number: $x")
end

# Standard entry function named exactly 'julia_main'
function (@main)(args::Vector{String})::Cint
    greet_user("Simon", 17)
    describe_item("Julia")
    describe_item(42)
    return 0
end
