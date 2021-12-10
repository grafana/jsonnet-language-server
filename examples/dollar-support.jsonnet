{
  // Support of $ within the same structure
  my_attribute: 'value',
  my_other_attribute: $.my_attribute,
}
+ {
  // Support of $ in merged structures
  from_merge: $.my_attribute,
}
