import React from "react";
import type { Props } from "./props.go";

export default function Dashboard(props: Props) {
  const [count, setCount] = React.useState(0);
  return (
    <main>
      <h1>Welcome to the Dashboard, {props.name}!</h1>
      <p>Count: {count}!!!!!!</p>
      <button onClick={() => setCount(count + 1)}>Increment</button>
    </main>
  );
}
