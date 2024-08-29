# ErrorSystem

## ErrorBar component

The error bar is a component that displays a list of error messages and next to
each error message is a button that allows the user to dismiss the error. If the
messages are dismissed, they are removed from the list. If there are no
messages, the error bar is hidden. It is placed in the layout component so it is
visible on every page and persists across navigation. This error bar component
will be at ui/src/lib/components/ErrorBar.svelte. The ErrorBar component has no
props of its own. It simply displays the errorMessages array contents if there
are any, and allows the user to dismiss the errors. It's a full-width bar at the
top of the page styled with tailwind. It will be included in the layout in
ui/src/routes/+layout.svelte as the first element within the main element. It
will have a light red background, no border, and the text will be dark red.
There will be no border radius on the edges.

## errorMessages in global store

The global store is at ui/src/lib/stores/global.ts. The errorMessages array is
used to store the error messages that are displayed in the ErrorBar component.
It is an array of objects with the following properties:

- `message`: The error message to display.
- `id`: The id of the error message.

The id is used to identify the error message when it is dismissed.

The errorMessages array is a writable store, so it can be updated by any
component. The global store is the only component that should update the
errorMessages array.

### Adding errors

The `addError` function is a function of the global store and is used to add an
error to the errorMessages array. It takes a single string argument representing
the error message. A unique id is generated for the error message and added to
the array.

### Dismissing errors

The `dismissError` function is a function of the global store and is used to
dismiss an error from the errorMessages array. It takes a single string argument
representing the id of the error to dismiss. The error is removed from the
array.
