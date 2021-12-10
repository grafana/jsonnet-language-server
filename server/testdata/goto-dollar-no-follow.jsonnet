local imported = import 'goto-dollar-simple.jsonnet';

{
  test: imported.attribute.sub,
}
+ {
  // Go to def on $.test should go to the attribute in this file, not follow through to the imported file.
  test2: $.test,
}
