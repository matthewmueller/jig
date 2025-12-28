export type Props = {
  error: Error;
};

export default function (props: Props) {
  return (
    <div style={{ color: "red" }}>
      Custom error page: {props.error.message}!
    </div>
  );
}
