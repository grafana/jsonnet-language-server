local file = (import "goto-import-intermediary.libsonnet").otherFile;
{
    foo: file.foo
}