# Namespaces in Environment Variables

In this example, we set up two separate HTTP servers.
Their configurations are read from the environment,
and using namespacing for the other server's variables,
we can use the same configuration struct for both.
