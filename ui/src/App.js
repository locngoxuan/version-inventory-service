import React, { useState, useEffect } from "react";
import "./App.css";
import { Container, Row, Card, Badge } from "react-bootstrap";
import Moment from "react-moment";
// import { ArrowRightIcon } from "@primer/octicons-react";

function App() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState(null);
  useEffect(() => {
    fetch("/api/v1/repos")
      .then((response) => {
        if (response.ok) {
          return response.json();
        }
        throw response;
      })
      .then((data) => {
        setData(data);
      })
      .catch((err) => {
        console.log("Error fetching: ", err);
        setErr(err);
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  if (loading) return "Loading...";
  if (err) return "Error!";

  return (
    <Container fluid style={{ padding: "0.5rem" }}>
      <Row style={{ margin: "0px", padding: "0px" }}>
        {data.map((e, i) => {
          console.log(e, i);
          return (
            <Card
              className="col-3 shadow p-2 bg-white rounded"
              style={{ margin: "0.5rem", padding: "0px" }}
            >
              <Card.Body>
                <Card.Title>
                  <span style={{ fontSize: "1.1rem" }}>{e.repo_id}</span>
                  {/* <div className="view-detail" style={{ float: "right" }}>
                <ArrowRightIcon size={16} />
              </div> */}
                </Card.Title>
                <Card.Subtitle className="mb-2">
                  <Badge bg="secondary" style={{ fontWeight: "normal" }}>
                    {e.namespace}
                  </Badge>{" "}
                </Card.Subtitle>
                <div>
                  <table className="latest-infor" style={{ width: "100%" }}>
                    <tbody>
                      <tr>
                        <td>Recent build</td>
                        <td className="right">
                          <Moment format="YYYY/MM/DD hh:mm">
                            {e.last_update}
                          </Moment>
                        </td>
                      </tr>
                      <tr>
                        <td>Development version</td>
                        <td className="right">{e.last_development}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </Card.Body>
              <Card.Footer style={{ background: "transparent", border: "0px" }}>
                <div className="statistic">
                  <div style={{ float: "left" }}>
                    <span className="release number">{e.cnt_release}</span>{" "}
                    Releases
                  </div>
                  <div style={{ float: "left", marginLeft: "1rem" }}>
                    <span className="patch number">{e.cnt_patch}</span> Patches
                  </div>
                </div>
              </Card.Footer>
            </Card>
          );
        })}
      </Row>
    </Container>
  );
}

export default App;
