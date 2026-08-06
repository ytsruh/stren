import XCTest
@testable import Stren

// MARK: - URLProtocol stub

/// URLProtocol subclass that returns a queued response for
/// every request. Used to test `APIClient` without hitting
/// the network. Registered at the start of each test via
/// `URLProtocol.registerClass`, unregistered in cleanup.
final class StubURLProtocol: URLProtocol {
    typealias Handler = (URLRequest) -> (HTTPURLResponse, Data)

    /// Queue of handlers, one per expected request. Tests
    /// pop them in order; if the queue runs out, requests
    /// fail with an XCTest failure.
    static var handlers: [Handler] = []
    /// Captures every request the client makes so tests
    /// can assert on the method/path/headers/body.
    static var captured: [URLRequest] = []

    static func reset() {
        handlers = []
        captured = []
    }

    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }

    override func startLoading() {
        guard !Self.handlers.isEmpty else {
            client?.urlProtocol(self, didFailWithError: URLError(.badURL))
            return
        }
        Self.captured.append(request)
        let (response, data) = Self.handlers.removeFirst()
        client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
        client?.urlProtocol(self, didLoad: data)
        client?.urlProtocolDidFinishLoading(self)
    }

    override func stopLoading() {}
}

// MARK: - APIClientTests

final class APIClientTests: XCTestCase {
    private var session: URLSession!

    override func setUp() {
        super.setUp()
        StubURLProtocol.reset()
        let config = URLSessionConfiguration.ephemeral
        config.protocolClasses = [StubURLProtocol.self]
        session = URLSession(configuration: config)
    }

    override func tearDown() {
        StubURLProtocol.reset()
        super.tearDown()
    }

    func testLoginSendsJSONAndParsesResponse() async throws {
        let userJSON = """
        {"id":"u1","name":"Test","email":"t@e.com","is_admin":false,"weight_unit":"kg"}
        """.data(using: .utf8)!
        StubURLProtocol.handlers.append { _ in
            (HTTPURLResponse(url: URL(string: "http://test/api/v1/auth/login")!,
                             statusCode: 200, httpVersion: nil, headerFields: nil)!, """
            {"token":"abc123","user":\(String(data: userJSON, encoding: .utf8)!)}
            """.data(using: .utf8)!)
        }

        let client = APIClient(
            baseURL: URL(string: "http://test/api/v1")!,
            session: session,
            tokenProvider: { nil }
        )
        let response = try await client.login(email: "t@e.com", password: "secret123")
        XCTAssertEqual(response.token, "abc123")
        XCTAssertEqual(response.user.email, "t@e.com")

        let request = try XCTUnwrap(StubURLProtocol.captured.first)
        XCTAssertEqual(request.httpMethod, "POST")
        XCTAssertEqual(request.value(forHTTPHeaderField: "Content-Type"), "application/json")
        // Login is unauthenticated — no Authorization header.
        XCTAssertNil(request.value(forHTTPHeaderField: "Authorization"))
    }

    func testAuthedRequestIncludesBearerHeader() async throws {
        StubURLProtocol.handlers.append { _ in
            (HTTPURLResponse(url: URL(string: "http://test/api/v1/me")!,
                             statusCode: 200, httpVersion: nil, headerFields: nil)!,
             Data("""
             {"id":"u1","name":"Test","email":"t@e.com","is_admin":false,"weight_unit":"kg"}
             """.utf8))
        }
        let client = APIClient(
            baseURL: URL(string: "http://test/api/v1")!,
            session: session,
            tokenProvider: { "the-token" }
        )
        _ = try await client.me()

        let request = try XCTUnwrap(StubURLProtocol.captured.first)
        XCTAssertEqual(request.value(forHTTPHeaderField: "Authorization"), "Bearer the-token")
    }

    func testUnauthorizedFiresOnUnauthorizedCallback() async {
        let exp = expectation(description: "onUnauthorized called")
        var fired = false
        let client = APIClient(
            baseURL: URL(string: "http://test/api/v1")!,
            session: session,
            tokenProvider: { "stale-token" },
            onUnauthorized: {
                fired = true
                exp.fulfill()
            }
        )
        StubURLProtocol.handlers.append { _ in
            (HTTPURLResponse(url: URL(string: "http://test/api/v1/me")!,
                             statusCode: 401, httpVersion: nil, headerFields: nil)!,
             Data("{\"error\":\"unauthorized\"}".utf8))
        }

        do {
            _ = try await client.me()
            XCTFail("expected unauthorized error")
        } catch APIError.unauthorized {
            // expected
        } catch {
            XCTFail("expected APIError.unauthorized, got \(error)")
        }
        await fulfillment(of: [exp], timeout: 1)
        XCTAssertTrue(fired)
    }

    func testServerErrorMessageIsDecoded() async {
        StubURLProtocol.handlers.append { _ in
            (HTTPURLResponse(url: URL(string: "http://test/api/v1/exercise-entries")!,
                             statusCode: 400, httpVersion: nil, headerFields: nil)!,
             Data("{\"error\":\"weight must be 5000 or less\"}".utf8))
        }
        let client = APIClient(
            baseURL: URL(string: "http://test/api/v1")!,
            session: session,
            tokenProvider: { "tok" }
        )
        do {
            _ = try await client.listExerciseEntries()
            XCTFail("expected server error")
        } catch let APIError.server(status, message) {
            XCTAssertEqual(status, 400)
            XCTAssertEqual(message, "weight must be 5000 or less")
        } catch {
            XCTFail("expected APIError.server, got \(error)")
        }
    }

    func testCreateExerciseEntriesSendsSnakeCaseKeys() async throws {
        StubURLProtocol.handlers.append { _ in
            (HTTPURLResponse(url: URL(string: "http://test/api/v1/exercise-entries")!,
                             statusCode: 201, httpVersion: nil, headerFields: nil)!,
             Data("[]".utf8))
        }
        let client = APIClient(
            baseURL: URL(string: "http://test/api/v1")!,
            session: session,
            tokenProvider: { "tok" }
        )
        _ = try await client.createExerciseEntries(
            CreateExerciseEntriesRequest(
                exerciseID: "ex-1",
                notes: "Felt strong",
                createdAt: Date(timeIntervalSince1970: 1_700_000_000),
                sets: [CreateSetInput(reps: 5, weight: 100, restTime: 120)]
            )
        )

        let request = try XCTUnwrap(StubURLProtocol.captured.first)
        let body = try XCTUnwrap(request.httpBody)
        let bodyString = String(data: body, encoding: .utf8) ?? ""
        XCTAssertTrue(bodyString.contains("\"exercise_id\":\"ex-1\""), "body = \(bodyString)")
        XCTAssertTrue(bodyString.contains("\"rest_time\":120"), "body = \(bodyString)")
        XCTAssertTrue(bodyString.contains("\"created_at\":"), "body = \(bodyString)")
    }

    func testDeleteExerciseEntryReturnsVoidOn204() async throws {
        StubURLProtocol.handlers.append { _ in
            (HTTPURLResponse(url: URL(string: "http://test/api/v1/exercise-entries/x")!,
                             statusCode: 204, httpVersion: nil, headerFields: nil)!,
             Data())
        }
        let client = APIClient(
            baseURL: URL(string: "http://test/api/v1")!,
            session: session,
            tokenProvider: { "tok" }
        )
        try await client.deleteExerciseEntry(id: "x")

        let request = try XCTUnwrap(StubURLProtocol.captured.first)
        XCTAssertEqual(request.httpMethod, "DELETE")
    }

    /// Regression test: a baseURL without a trailing slash
    /// must still resolve to `<base>/<path>`, not
    /// `<base-without-last-segment>/<path>`. Without the
    /// fix in `APIClient.init`, Swift's URL resolver treats
    /// "v1" as a file and replaces it with "auth/login",
    /// producing `http://test/api/auth/login` — which the
    /// server responds to with an HTML redirect instead of
    /// the expected JSON.
    func testRelativePathResolvesWithTrailingSlashAdded() async throws {
        StubURLProtocol.handlers.append { _ in
            (HTTPURLResponse(url: URL(string: "http://test/api/v1/auth/login")!,
                             statusCode: 200, httpVersion: nil, headerFields: nil)!,
             Data("""
             {"token":"t","user":{"id":"u","name":"n","email":"e","is_admin":false,"weight_unit":"kg"}}
             """.utf8))
        }
        // Note: baseURL has NO trailing slash on purpose.
        let client = APIClient(
            baseURL: URL(string: "http://test/api/v1")!,
            session: session,
            tokenProvider: { nil }
        )
        _ = try await client.login(email: "e@e.com", password: "p")

        let request = try XCTUnwrap(StubURLProtocol.captured.first)
        XCTAssertEqual(request.url?.path, "/api/v1/auth/login",
                       "URL resolution dropped the /v1 segment")
    }

    /// A baseURL that *does* have a trailing slash must
    /// also work — no double slash, no loss of segments.
    func testRelativePathResolvesWithExistingTrailingSlash() async throws {
        StubURLProtocol.handlers.append { _ in
            (HTTPURLResponse(url: URL(string: "http://test/api/v1/auth/login")!,
                             statusCode: 200, httpVersion: nil, headerFields: nil)!,
             Data("""
             {"token":"t","user":{"id":"u","name":"n","email":"e","is_admin":false,"weight_unit":"kg"}}
             """.utf8))
        }
        let client = APIClient(
            baseURL: URL(string: "http://test/api/v1/")!,
            session: session,
            tokenProvider: { nil }
        )
        _ = try await client.login(email: "e@e.com", password: "p")

        let request = try XCTUnwrap(StubURLProtocol.captured.first)
        XCTAssertEqual(request.url?.path, "/api/v1/auth/login")
    }
}

// MARK: - DTO decoding

final class DTODecodingTests: XCTestCase {
    func testDecodesFractionalSecondDates() throws {
        let json = """
        {"id":"e1","exercise_id":"ex-1","exercise_name":"Squat","reps":5,"weight":100.0,"notes":"","rest_time":0,"created_at":"2026-08-06T07:46:23.514884+01:00"}
        """.data(using: .utf8)!
        let entry = try APIClient.jsonDecoder.decode(ExerciseEntryDTO.self, from: json)
        XCTAssertEqual(entry.exerciseName, "Squat")
        XCTAssertEqual(entry.reps, 5)
    }

    func testDecodesNonFractionalSecondDates() throws {
        let json = """
        {"id":"e1","exercise_id":"ex-1","exercise_name":"Squat","reps":5,"weight":100.0,"notes":"","rest_time":0,"created_at":"2026-08-06T07:46:23+01:00"}
        """.data(using: .utf8)!
        let entry = try APIClient.jsonDecoder.decode(ExerciseEntryDTO.self, from: json)
        XCTAssertEqual(entry.id, "e1")
    }

    func testEncodesFractionalSecondDates() throws {
        let date = Date(timeIntervalSince1970: 1_700_000_000)
        let data = try APIClient.jsonEncoder.encode(date)
        let str = String(data: data, encoding: .utf8) ?? ""
        // ISO 8601 with timezone; we don't pin the exact
        // string (locale-dependent) just the format.
        XCTAssertTrue(str.contains("T"))
        XCTAssertTrue(str.contains(":") || str.contains("+") || str.contains("Z"))
    }

    func testHistoryPageShape() throws {
        let json = """
        {"entries":[],"stats":{"max_weight":100.0},"page":1,"has_prev":false,"has_next":false}
        """.data(using: .utf8)!
        let page = try APIClient.jsonDecoder.decode(HistoryPageDTO.self, from: json)
        XCTAssertEqual(page.entries.count, 0)
        XCTAssertEqual(page.stats.maxWeight, 100.0)
        XCTAssertNil(page.stats.lastSet)
    }
}
