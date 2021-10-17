// Go to definition will jump to the imported file.
// Try jumping from the import statement or the file name.
local example = import 'example.libsonnet';
// It also works with importstr.
local str = import 'example.txt';

// Go to definition knows where the '$' references.
// Try jump from each usage of '$'.
// Unfortunately, it is not yet able to jump to the field within the referenced object.
local obj = {
  y:: 25,
  a: $.y,
};
obj {
  z:: 26,
  b: $.z,

  // Go to definition knows which local is relevant.
  // Try jumping from each usage of 'c'.
  // Did the outcome match your mental model?
  local c = 1,
  c:
    [c]
    + local c = 2;
    [
      local c = 3;
      c,
      c,
    ],

  // Go to definition knows where the super object begins.
  // Try jumping from the usage of 'super'.
  // Unfortunately, it is not yet able to jump to the field within the super object.
  d: super.a,
}
