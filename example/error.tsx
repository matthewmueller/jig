export type Props = {
  error: Error
}

export default function (props: Props) {
  return <div style={{ color: "red" }}>{props.error.message}!</div>
}
