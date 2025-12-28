import React from "react";

export type Props = {
  name: string;
};

export default function Index(props: Props) {
  const [count, setCount] = React.useState(0);
  return (
    <main>
      <head>
        <title>Hello world</title>
        <meta charSet="UTF-8" />
        <meta name="cool" content={"story"} />
        {typeof window === "undefined" ? null : (
          <meta name="browser" content={"only"} />
        )}
      </head>
      <h1>Hello, {props.name}!!</h1>
      <Header2 />
      <p>Count: {count}</p>
      <button onClick={() => setCount(count + 1)}>Increment</button>
      <style jsx>{`
        h1,
        h2 {
          color: green;
        }
      `}</style>
    </main>
  );
}

function Header2() {
  return <h2>This is a header 2</h2>;
}
