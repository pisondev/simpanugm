import numpy as np

from tests.conftest import TOKEN


def test_cluster_requires_token(client):
    res = client.post("/v1/cluster", json={"items": [{"id": "1", "vector": [1.0, 0.0]}]})
    assert res.status_code == 401


def test_cluster_empty_items_rejected(client):
    res = client.post(
        "/v1/cluster",
        headers={"Authorization": f"Bearer {TOKEN}"},
        json={"items": []},
    )
    assert res.status_code == 400
    assert res.json()["code"] == "invalid_request"


def test_cluster_dimension_mismatch_rejected(client):
    res = client.post(
        "/v1/cluster",
        headers={"Authorization": f"Bearer {TOKEN}"},
        json={
            "items": [
                {"id": "a", "vector": [1.0, 0.0]},
                {"id": "b", "vector": [1.0, 0.0, 0.5]},
            ]
        },
    )
    assert res.status_code == 400
    assert res.json()["code"] == "invalid_request"


def test_cluster_fewer_than_min_cluster_size(client):
    res = client.post(
        "/v1/cluster",
        headers={"Authorization": f"Bearer {TOKEN}"},
        json={
            "items": [{"id": "a", "vector": [1.0, 0.0]}],
            "min_cluster_size": 2,
        },
    )
    assert res.status_code == 200
    data = res.json()
    assert len(data["labels"]) == 1
    assert data["labels"][0]["id"] == "a"
    assert data["labels"][0]["cluster"] == -1
    assert data["labels"][0]["probability"] == 0.0
    assert data["version"].startswith("sklearn-hdbscan-")


def test_cluster_synthetic_groups(client):
    np.random.seed(42)
    # Two well-separated clusters in 8 dimensions
    c1 = np.random.normal([1.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0], 0.02, (10, 8))
    c2 = np.random.normal([0.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0], 0.02, (10, 8))
    items = []
    for i in range(10):
        items.append({"id": f"c1_{i}", "vector": c1[i].tolist()})
    for i in range(10):
        items.append({"id": f"c2_{i}", "vector": c2[i].tolist()})

    res = client.post(
        "/v1/cluster",
        headers={"Authorization": f"Bearer {TOKEN}"},
        json={
            "items": items,
            "min_cluster_size": 3,
            "min_samples": 1,
        },
    )
    assert res.status_code == 200
    data = res.json()
    assert len(data["labels"]) == 20
    assert data["version"].startswith("sklearn-hdbscan-")

    labels_map = {item["id"]: item["cluster"] for item in data["labels"]}
    # Check that items in c1 share a cluster and items in c2 share a distinct cluster
    c1_clusters = {labels_map[f"c1_{i}"] for i in range(10) if labels_map[f"c1_{i}"] != -1}
    c2_clusters = {labels_map[f"c2_{i}"] for i in range(10) if labels_map[f"c2_{i}"] != -1}
    assert len(c1_clusters) == 1
    assert len(c2_clusters) == 1
    assert c1_clusters != c2_clusters
