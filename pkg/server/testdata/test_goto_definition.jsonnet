local myvar = 2;
local helper(x) = x * 2;

{
  b: myvar,
  a: std.max(x=myvar),
} + {
  c: helper(myvar),
}
