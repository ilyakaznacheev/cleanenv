# Custom value setter

In this example, a custom function is used to set a value in the configurations struct.

The configuration struct has a field of type ```roles```, which has a method ```SetValue```
defined on it. After fetching configuration values from environment variables, this method is 
executed.

The ```ROLES``` environment variable will have the value ```admin owner member``` as a single
string, but the field requires it as an array of strings. The ```SetValue``` method will split
the string and add each value into the roles field.